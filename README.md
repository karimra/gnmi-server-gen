# gnmi-server-gen

A simple gNMI server traffic generator
```
  -a, --address string              server address
  -c, --config string               config file
  -i, --interval duration           sample interval (default 1s)
  -n, --num-servers int             number of servers (default 1)
  -p, --port uint16                 gnmi servers start port (default 57400)
      --prometheus-address string   prometheus server address
  -r, --rate int                    number of updates per interval (default 1)
      --tls-ca string               TLS CA path
      --tls-cert string             TLS certificate path
      --tls-key string              TLS key path
  -v, --version                     print version
