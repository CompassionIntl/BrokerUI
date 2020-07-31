package configuration

type BrokerConfiguration struct {
	Name string
	Type string
	URL  string
	User string
	Pass string
	All  map[string]string
}
