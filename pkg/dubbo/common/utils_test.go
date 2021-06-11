package common

import "testing"

func Test_parseNacosURL(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		host    string
		port    uint64
		wantErr bool
	}{
		{
			name:    "valid addr",
			addr:    "nacos.dubbo:8848",
			host:    "nacos.dubbo",
			port:    8848,
			wantErr: false,
		},
		{
			name:    "invalid addr",
			addr:    "nacos.dubbo",
			host:    "",
			port:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := parseNacosURL(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNacosURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if host != tt.host {
				t.Errorf("parseNacosURL() got = %v, want %v", host, tt.host)
			}
			if port != tt.port {
				t.Errorf("parseNacosURL() got1 = %v, want %v", port, tt.port)
			}
		})
	}
}
