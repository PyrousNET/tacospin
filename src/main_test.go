package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_getWindSpeed(t *testing.T) {
	type args struct {
		server *httptest.Server
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{
			name: "valid wind speed",
			args: args{
				server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`{ "entries": [ { "wind_speed": "5.5" } ] }`))
					if err != nil {
						return
					}
				})),
			},
			want: 5.5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer tt.args.server.Close() // Ensure the server is closed after the test
			got, err := getWindSpeed(tt.args.server.URL + "/weather")
			if (err != nil) != tt.wantErr {
				t.Errorf("getWindSpeed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getWindSpeed() got = %v, want %v", got, tt.want)
			}
		})
	}
}
