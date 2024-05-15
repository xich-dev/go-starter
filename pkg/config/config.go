package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/xich-dev/go-starter/pkg/logger"
	"gopkg.in/yaml.v3"
)

var log = logger.NewLogAgent("config")

type Pg struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Db       string `yaml:"db"`
	Port     int    `yaml:"port"`
	// the path of directory to store migration files
	Migration string `yaml:"migration"`
}

type TecentCloudSMS struct {
	Enable        bool   `yaml:"enable"`
	SecretKey     string `yaml:"secretkey"`
	SecretId      string `yaml:"secretid"`
	SmsId         string `yaml:"id"`
	SmsSigName    string `yaml:"smssigname"`
	SmsTemplateId string `yaml:"smstemplateid"`
}

type Jwt struct {
	Secret string `yaml:"secret"`
}

type Config struct {
	Port  int            `yaml:"port,omitempty"`
	TCSMS TecentCloudSMS `yaml:"tcsms,omitempty"`
	Debug bool           `yaml:"debug,omitempty"`

	Jwt Jwt `yaml:"jwt,omitempty"`
	Pg  Pg  `yaml:"pg,omitempty"`
}

func NewConfig() (*Config, error) {
	c := &Config{}
	if err := fetchConfig("config.yaml", c); err != nil {
		return nil, errors.Wrap(err, "failed to fetch config")
	}
	if len(c.Pg.Migration) == 0 {
		c.Pg.Migration = "migrations"
	}
	return c, nil
}

var prefix = "XICFG_"

func fetchConfig(configPath string, cfg interface{}) error {
	yamlRaw, err := readConfigFromPathAndEnv(configPath)
	if err != nil {
		return errors.Wrap(err, "failed to read and patch config")
	}
	return marshallRawYAML(yamlRaw, cfg)
}

func marshallRawYAML(yamlRaw []byte, cfg interface{}) error {
	err := yaml.Unmarshal(yamlRaw, cfg)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal yaml config %v", yamlRaw)
	}
	return nil
}

func readConfigFromPathAndEnv(configPath string) ([]byte, error) {
	config, err := readFromConfigFile(configPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read config from %v", configPath)
	}
	log.Infof("from config file: %+v", config)

	configEnv := readFromConfigEnv()
	log.Infof("from config env: %+v", configEnv)

	if err := patchConfigMap(configEnv, config); err != nil {
		return nil, errors.Wrap(err, "failed to patch config env to config file")
	}
	yamlRaw, err := yaml.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "yaml marshal error")
	}
	return yamlRaw, nil
}

// patchConfigMap partially validates that both patch and base, then merge patch into base.
func patchConfigMap(patch, base map[string]any) error {
	if err := patchMap(base, patch); err != nil {
		return errors.Wrap(err, "failed to patch to config file")
	}
	log.Infof("final config: %+v", base)
	return nil
}

func readFromConfigFile(configPath string) (map[string]any, error) {
	config := map[string]any{}

	_, err := os.Stat(configPath)
	if err != nil {
		log.Warnf("cannot read config file: %+v", err)
		return config, nil
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return config, errors.Wrap(err, "failed to read config file")
	}

	err = yaml.Unmarshal(raw, &config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config file")
	}

	return config, nil
}

func readFromConfigEnv() map[string]any {
	envCfg := map[string]any{}
	for _, v := range os.Environ() {
		if strings.HasPrefix(v, prefix) {
			key := strings.Split(v, "=")[0]
			value := v[len(key)+1:]
			key = strings.ToLower(strings.Replace(key, prefix, "", 1))
			parseEnvConfig(envCfg, key, value)
		}
	}
	return envCfg
}

// parseEnvConfig turn an environment variable to a map
// by convention, the env key has pattern A_B_C with each yaml config key separated by _
// calling function with key=MGMT_LOG_LEVEL and value=INFO
// should update curCfg to {MGMT: {LOG: [LEVEL: INFO}}}.
func parseEnvConfig(curCfg map[string]any, key string, value string) {
	i := strings.Index(key, "_")
	if i == -1 {
		if intVal, err := strconv.Atoi(value); err == nil {
			curCfg[key] = intVal
		} else if boolVal, err := strconv.ParseBool(value); err == nil {
			curCfg[key] = boolVal
		} else {
			curCfg[key] = value
		}
	} else {
		thisKey := key[:i]
		if _, ok := curCfg[thisKey]; !ok {
			curCfg[thisKey] = map[string]any{}
		}
		parseEnvConfig(curCfg[thisKey].(map[string]any), key[i+1:], value)
	}
}

func patchMap(o map[string]any, p map[string]any) error {
	for k := range p {
		if _, ok := o[k]; ok { // if o has the same key
			if _, ok := o[k].(map[string]any); ok {
				if _, ok := p[k].(map[string]any); !ok {
					return errors.Errorf("%s of %s is not a map[string]any", k, p)
				}
				// o[k] and p[k] are both map
				if err := patchMap(o[k].(map[string]any), p[k].(map[string]any)); err != nil {
					return err
				}
			} else { // both are values
				o[k] = p[k]
			}
		} else { // o does not have this key
			o[k] = p[k]
		}
	}
	return nil
}
