# Deprecation Notice
Authentik now has this feature natively, although without avatar support.

## Use case
Provides webfinger discovery backed by authentik.

An example of this is setting up a tailscale account with self hosted sso through authentik.

## Config

`config.toml` example:
```toml
Host = ":8080"
AuthentikHost = "authentik.example.com"
Token = "changeme"
UserAgent = "tfowl/authentik-webfinger (https://github.com/T-Fowl/authentik-webfinger)"
AuthentikApplication = "tailscale"
```

## Reverse Proxy Usage

### Caddy
```
:443 {
    handle /.well-known/webfinger {
         reverse_proxy localhost:8080    
    }
    
    handle {
        reverse_proxy other_backend_servers
    }
}
```

### Nginx
```
location /.well-known/webfinger {
    proxy_pass http://localhost:8080;
}
location / {
    ...
}
```