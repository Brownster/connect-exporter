package maas

// ServiceFeed represents configuration for a single RSS/Atom feed service
type ServiceFeed struct {
	Name     string `yaml:"name"`
	Provider string `yaml:"provider"`
	Customer string `yaml:"customer"`
	URL      string `yaml:"url"`
	Interval int    `yaml:"interval"`
}