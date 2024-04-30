package config

import (
	"bytes"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/alecthomas/units"
	"github.com/mcuadros/go-defaults"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"net/url"
	"os"
	"strings"
	"time"
)

var Mode string

var Version string

var Config Struct

type ServiceDescription struct {
	Type        ServiceType `toml:"type"`
	Bind        Address     `toml:"bind"`
	Certificate string      `toml:"certificate,omitempty"`
	Destination Address     `toml:"destination,omitempty"`
}

type Struct struct {
	Preflight struct {
		UploadCodec      PreflightUploadCodec `default:"auto" toml:"upload_codec"`
		UploadChunkSize  units.Base2Bytes     `default:"65536" toml:"upload_chunk_size"`
		AssumeNoInternet bool                 `default:"false" toml:"assume_no_internet"`
		CommandTimeout   time.Duration        `default:"10s" toml:"command_timeout"`
	}
	Transfer struct {
		Codec                   CodecType        `default:"hex" toml:"codec"`
		SignatureDetectorBuffer units.Base2Bytes `default:"10240" toml:"signature_detector_buffer"`
		Buffer                  units.Base2Bytes `default:"65536" toml:"buffer"`
		ConnectionTimeout       time.Duration    `default:"10s" toml:"connection_timeout"`
		RequestTimeout          time.Duration    `default:"10s" toml:"request_timeout"`
	}
	Log struct {
		File  string       `default:"unbound_ssh_$(mode).log" toml:"file"`
		Level logrus.Level `default:"4" toml:"level"`
	}
	Service []ServiceDescription `toml:"service"`
}

// ---------------------------------------------------------------------------

type PreflightUploadCodec string

const (
	Auto        PreflightUploadCodec = "auto"
	Base64      PreflightUploadCodec = "base64"
	GzipBase64  PreflightUploadCodec = "gzip+base64"
	Ascii85     PreflightUploadCodec = "ascii85"
	GzipAscii85 PreflightUploadCodec = "gzip+ascii85"
)

func (s *PreflightUploadCodec) UnmarshalText(text []byte) error {
	validValues := []PreflightUploadCodec{Auto, Base64, GzipBase64, Ascii85, GzipAscii85}
	serviceType := PreflightUploadCodec(text)
	if !lo.Contains(validValues, serviceType) {
		return fmt.Errorf("invalid preflight upload codec: %s", text)
	}
	*s = PreflightUploadCodec(text)
	return nil
}

// ---------------------------------------------------------------------------

type CodecType string

const (
	Hex   CodecType = "hex"
	Plain CodecType = "plain"
)

func (s *CodecType) UnmarshalText(text []byte) error {
	validValues := []CodecType{Hex, Plain}
	serviceType := CodecType(text)
	if !lo.Contains(validValues, serviceType) {
		return fmt.Errorf("invalid codec: %s", text)
	}
	*s = CodecType(text)
	return nil
}

// ---------------------------------------------------------------------------

type ServiceType string

const (
	EmbeddedWebdav    ServiceType = "embedded_webdav"
	EmbeddedSsh       ServiceType = "embedded_ssh"
	EmbeddedHttpProxy ServiceType = "embedded_http_proxy"
	PortForward       ServiceType = "port_forward"
	Echo              ServiceType = "echo"
)

func (s *ServiceType) UnmarshalText(text []byte) error {
	validValues := []ServiceType{EmbeddedWebdav, EmbeddedSsh, EmbeddedHttpProxy, PortForward, Echo}
	serviceType := ServiceType(text)
	if !lo.Contains(validValues, serviceType) {
		return fmt.Errorf("invalid service type: %s", text)
	}
	*s = ServiceType(text)
	return nil
}

// ---------------------------------------------------------------------------

type Address struct {
	network string
	string  string
}

func NewAddress(network, string string) Address {
	return Address{network: network, string: string}
}

func (addr *Address) Network() string {
	return addr.network
}

func (addr *Address) String() string {
	return addr.string
}

func (addr *Address) FullAddress() string {
	return fmt.Sprintf("%s://%s", addr.Network(), addr.String())
}

func (addr *Address) UnmarshalText(text []byte) error {
	parsed, err := url.Parse(ProcessString(string(text)))
	if err != nil {
		return err
	}

	addr.network = parsed.Scheme
	if strings.HasPrefix(parsed.Scheme, "unix") {
		addr.string = parsed.Path
	} else {
		addr.string = parsed.Host
	}
	return nil
}

func (addr *Address) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%s://%s", addr.Network(), addr.String())), nil
}

// ---------------------------------------------------------------------------

func (s *Struct) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		logrus.Errorf("error reading toml config file %s: %s", path, err.Error())
		return err
	}
	return s.LoadData(string(data))
}

func (s *Struct) LoadData(data string) error {
	_, err := toml.Decode(data, s)
	if err != nil {
		logrus.Errorf("error parsing toml config: %s", err.Error())
		return err
	}
	err = validateConfig()
	return err
}

func (s *Struct) Clone() *Struct {
	clone := Struct{}
	clone = Config
	return &clone
}

func (s *Struct) SaveData() string {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	encoder := toml.NewEncoder(buf)
	err := encoder.Encode(s)
	if err != nil {
		logrus.Errorf("error saving toml config: %s", err.Error())
		panic(err)
	}
	return buf.String()
}

func init() {
	defaults.SetDefaults(&Config)
}
