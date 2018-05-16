Webhook Proxy
=============

Proxies Github webhooks to internal servers.

## Getting started

### Build

```
make
```

### Creating an endpoint

```
base64EncodedSecret=$(echo -n "foobar" | base64)
curl -X POST -d '{"name": "receive-all-hook", "team": "my-team-name", "secret": "'$base64EncodedSecret'", "url": "http://internal-server.org/myapp"}' http://localhost:8080/new
```

The response will be something like:
```json
{
  "id":"368a1500082a071a7629c6ad704f7289e220fcc9",
  "name":"receive-all-hook",
  "team":"my-team-name",
  "url":"http://internal-server.org/myapp",
  "secret":"Zm9vYmFy",
  "proxy_url":"http://localhost:8080/hook/368a1500082a071a7629c6ad704f7289e220fcc9",
  "created_at":"2018-05-16T10:54:58.1838475Z"
}
```

Use `proxy_url` as webhook url when creating the webhook in GitHub and use the secret that you generated when 
creating the webhook proxy endpoint.

---

# Contact us

Code/project related questions can be sent to:

* David Steinsland, david.steinsland@nav.no
* André Roaldseth, andre.roaldseth@nav.no

## For NAV employees

We are also available on the slack channel #github-webhooks for internal communication.
