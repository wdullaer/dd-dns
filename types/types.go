package types

type DNSLabel struct {
	Name        string
	IP          string
	ContainerID []string
}

type DNSMapping struct {
	Name        string
	IP          string
	ContainerID string
}

func (mapping *DNSMapping) GetKey() []byte {
	return []byte(mapping.Name + mapping.IP)
}
