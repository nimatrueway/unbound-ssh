package listen

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/ascii85"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/sirupsen/logrus"
	stdio "io"
	"net/http"
	"os"
	"path"
)

func ReadFileAndUpload(ctx context.Context, shell *ShellExecutor, file string, filename string, encoding config.PreflightUploadCodec) error {
	reader, err := createFileReader(file)
	if err != nil {
		return err
	}
	if filename == "" {
		filename = path.Base(file)
	}

	return upload(ctx, shell, filename, reader, encoding)
}

func FetchUrlAndUpload(ctx context.Context, shell *ShellExecutor, url string, filename string, encoding config.PreflightUploadCodec) error {
	reader, err := createUrlReader(url)
	if err != nil {
		return err
	}

	return upload(ctx, shell, filename, reader, encoding)
}

func Upload(ctx context.Context, shell *ShellExecutor, content string, filename string, encoding config.PreflightUploadCodec) error {
	return upload(ctx, shell, filename, stdio.NopCloser(bytes.NewBufferString(content)), encoding)
}

func upload(ctx context.Context, shell *ShellExecutor, filename string, reader stdio.ReadCloser, encoding config.PreflightUploadCodec) error {
	// stage 0: tap reader to calculate size/hash
	size := uint64(0)
	hash := [32]byte{}
	reader = trackSizeAndHash(reader, &size, &hash)

	// stage 1: add gzip filter
	if encoding == config.GzipBase64 || encoding == config.GzipAscii85 {
		logrus.Debugf("will compress input in gzip")
		reader = applyEncoder(reader, "gzip", func(w stdio.Writer) stdio.WriteCloser {
			zipper, _ := gzip.NewWriterLevel(w, gzip.BestCompression)
			return zipper
		})
	}

	// stage 2: add base64/ascii85 filter
	if encoding == config.Base64 || encoding == config.GzipBase64 {
		reader = applyEncoder(reader, "base64", func(writer stdio.Writer) stdio.WriteCloser {
			return base64.NewEncoder(base64.StdEncoding, writer)
		})
	} else if encoding == config.Ascii85 || encoding == config.GzipAscii85 {
		reader = applyEncoder(reader, "ascii85", func(writer stdio.Writer) stdio.WriteCloser {
			return ascii85.NewEncoder(writer)
		})
	}

	// stage 3: transfer file content
	logrus.Debug("transferring file content")
	defer func() {
		err := reader.Close()
		if err != nil {
			logrus.Warnf("Close() failed on the reader: %s", err.Error())
		}
	}()
	err := writeFile(ctx, shell, filename, reader)
	if err != nil {
		logrus.Warnf("transferring file content failed: %s", err.Error())
		return err
	}
	logrus.Infof("all bytes successfully transferred.")

	// stage 4: decode the transferred file
	if encoding == config.Base64 || encoding == config.GzipBase64 {
		err = decodeBase64File(ctx, shell, filename)
	} else if encoding == config.Ascii85 || encoding == config.GzipAscii85 {
		err = decodeAscii85File(ctx, shell, filename)
	}
	if err != nil {
		return err
	}

	// stage 5: un-gzip the transferred file
	if encoding == config.GzipBase64 || encoding == config.GzipAscii85 {
		err = unGzipFile(ctx, shell, filename)
		if err != nil {
			return err
		}
	}

	// stage 6: verify size and hash of the uploaded file
	actualSize, actualHash, err := fetchSizeAnsHash(ctx, shell, filename)
	if err != nil {
		return err
	}
	if size != actualSize {
		return fmt.Errorf("size mismatch: expected %d received %d", size, actualSize)
	}
	if hash != actualHash {
		return fmt.Errorf("hash mismatch: expected %x received %x", hash, actualHash)
	}

	return nil
}

func createUrlReader(url string) (stdio.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET %s failed: %s", url, err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		errorBody, err := stdio.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("GET %s responded with code: %d headers: %#v body: error(%s)", url, resp.StatusCode, resp.Header, err.Error())
		} else {
			return nil, fmt.Errorf("GET %s responded with code: %d headers: %#v body: %s", url, resp.StatusCode, resp.Header, string(errorBody))
		}
	}

	return resp.Body, nil
}

func createFileReader(file string) (stdio.ReadCloser, error) {
	return os.Open(file)
}

func writeFile(ctx context.Context, shell *ShellExecutor, filename string, data stdio.ReadCloser) error {
	// upon exit clean up temporary chunks
	defer func() {
		_, closeErr := shell.Execute(context.Background(), fmt.Sprintf(`rm %s_chunk_*.tmp`, filename), nil)
		if closeErr != nil {
			logrus.Warnf("failed to clean up temporary chunks: %s", closeErr.Error())
		}
	}()

	// capture current stty settings
	res, err := shell.Execute(ctx, `stty -g`, nil)
	if err != nil {
		return err
	}

	// upon exit restore terminal settings
	defer func() {
		_, closeErr := shell.Execute(context.Background(), fmt.Sprintf(`stty "%s"`, res.Output), nil)
		if closeErr != nil {
			logrus.Warnf("failed to restore terminal settings: %s", closeErr.Error())
		}
	}()

	// set terminal to raw mode, to avoid tty echoing and buffering limitation
	_, err = shell.Execute(ctx, `stty raw opost -echo`, nil)
	if err != nil {
		return err
	}

	// make data read interruptible through context cancellation
	ctxData := core.NewContextReader(data)
	data = ctxData.BindTo(ctx)

	// split file content into smaller temporary chunk files and transfer them
	var MaxChunkSize = config.Config.Preflight.UploadChunkSize
	buf := make([]byte, MaxChunkSize)
	for chunk := 0; true; chunk++ {
		nRead, err := stdio.ReadAtLeast(data, buf, len(buf))
		if err == stdio.EOF || errors.Is(err, stdio.ErrUnexpectedEOF) {
			err = nil
		}
		if err != nil {
			return err
		}
		if nRead == 0 {
			break
		}

		chunkName := fmt.Sprintf(`%s_chunk_%07d.tmp`, filename, chunk)
		// this is intentionally non-cancellable, as we need to ensure that the chunk writing is completed
		// otherwise "dd" running in raw-mode will freeze the terminal
		_, err = shell.Execute(context.Background(), fmt.Sprintf(`dd bs=%d count=1 iflag=fullblock >%s`, nRead, chunkName), buf[:nRead])
		if err != nil {
			return err
		}
	}

	// concatenate all chunks into the final file
	_, err = shell.Execute(ctx, fmt.Sprintf(`cat %[1]s_chunk_*.tmp > %[1]s`, filename), nil)

	return err
}

// trackSizeAndHash wraps the reader to calculate the size and hash of the content
func trackSizeAndHash(reader stdio.ReadCloser, size *uint64, hash *[32]byte) stdio.ReadCloser {
	hasher := sha256.New()
	pipeR, pipeW := stdio.Pipe()
	go func() {
		defer func() {
			err := reader.Close()
			if err != nil {
				logrus.Warnf("Close() failed on the underlying reader of size/hash tracked: %s", err.Error())
			}
		}()

		n, err := stdio.Copy(stdio.MultiWriter(hasher, pipeW), reader)
		if err != nil {
			logrus.Warnf("size/hash tracker failed: %s", err.Error())
		}
		err = pipeW.CloseWithError(stdio.EOF)
		if err != nil {
			logrus.Warnf("pipe close failed on size/hash tracker: %s", err.Error())
		}

		*size = uint64(n)
		copy(hash[:], hasher.Sum([]byte{}))
	}()
	return pipeR
}

func applyEncoder(reader stdio.ReadCloser, name string, encoder func(stdio.Writer) stdio.WriteCloser) stdio.ReadCloser {
	pipeR, pipeW := stdio.Pipe()
	encoderW := encoder(pipeW)
	go func() {
		// close the underlying reader upon exhaustion of the content
		defer func() {
			err := reader.Close()
			if err != nil {
				logrus.Warnf("Close() failed on the underlying reader of encoder %s: %s", name, err.Error())
			}
		}()

		// copy the reader content to the encoder, which will encode it and write to the writer pipe,
		// and that can be read through the pipe reader
		_, err := stdio.Copy(encoderW, reader)
		if err != nil {
			logrus.Warnf("encoder %s failed: %s", name, err.Error())
		}

		// close the encoder so that it can flush the remaining data
		flushErr := encoderW.Close()
		if flushErr != nil {
			logrus.Warnf("Close() failed on encoder %s: %s", name, flushErr.Error())
		}

		// close the writer pipe, so that reader pipe will close as well
		var pipeCloseErr error
		if err == nil && flushErr == nil {
			pipeCloseErr = pipeW.CloseWithError(stdio.EOF)
		} else if err != nil {
			pipeCloseErr = pipeW.CloseWithError(err)
		} else if flushErr != nil {
			pipeCloseErr = pipeW.CloseWithError(flushErr)
		}
		if pipeCloseErr != nil {
			logrus.Warnf("Close() failed on the writer pipe of encoder %s: %s", name, pipeCloseErr.Error())
		}
	}()
	return pipeR
}

func ensureDir(ctx context.Context, shell *ShellExecutor, filename string) (err error) {
	_, err = shell.Execute(ctx, fmt.Sprintf(`mkdir -p "%s"`, path.Dir(filename)), nil)
	return err
}

func decodeBase64File(ctx context.Context, shell *ShellExecutor, filename string) (err error) {
	_, err = shell.Execute(ctx, fmt.Sprintf(`base64 -d -i "%[1]s" > "%[1]s_decoded" && mv "%[1]s_decoded" "%[1]s"`, filename), nil)
	return err
}

func decodeAscii85File(ctx context.Context, shell *ShellExecutor, filename string) (err error) {
	const script = `python3 -c "import sys, base64; sys.stdout.buffer.write(base64.a85decode(open(sys.argv[1], 'rb').read()))"`
	_, err = shell.Execute(ctx, fmt.Sprintf(`%[2]s "%[1]s" > "%[1]s_decoded" && mv "%[1]s_decoded" "%[1]s"`, filename, script), nil)
	return err
}

func unGzipFile(ctx context.Context, shell *ShellExecutor, filename string) (err error) {
	_, err = shell.Execute(ctx, fmt.Sprintf(`gunzip -c -k "%[1]s" > "%[1]s_decompressed" && mv "%[1]s_decompressed" "%[1]s"`, filename), nil)
	return err
}

func fetchSizeAnsHash(ctx context.Context, shell *ShellExecutor, filename string) (size uint64, hash [32]byte, err error) {
	collector, err := Execute[*GetFileSizeAndHash](ctx, shell, nil)(GenerateGetFileSizeAndHash(filename))
	if err != nil {
		return 0, [32]byte{}, err
	}

	return collector.Size, collector.Hash, nil
}
