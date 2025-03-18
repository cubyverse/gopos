package config

type Config struct {
	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`
	Email struct {
		Endpoint   string `yaml:"endpoint"`
		AccessKey  string `yaml:"access_key"`
		SenderMail string `yaml:"sender_mail"`
	} `yaml:"email"`
	Session struct {
		Key string `yaml:"key"`
	} `yaml:"session"`
}
