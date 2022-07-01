# Mesh Crawler
A Python based tool to test connectivity inside a service mesh.

## Usage

### GET

A GET request to this app will just send you a reponse that echoes its incoming request headers.

**Response**
```
{
    "myIncomingRequestheaders": {
        "Accept": "*/*",
        "Accept-Encoding": "gzip, deflate, br",
        "Connection": "keep-alive",
        "Content-Length": "165",
        "Content-Type": "text/plain",
        "Host": "mesh-crawler-bob_apps-1-default_svc_8080.mesh",
        "Postman-Token": "014addf0-66dd-4edb-90fd-ae37a14d3f8a",
        "User-Agent": "PostmanRuntime/7.29.0",
        "X-Forwarded-For": "192.168.3.213",
        "X-Forwarded-Host": "bob.mesh-zone-a.k8s.mschnkvld.lab",
        "X-Forwarded-Path": "/",
        "X-Forwarded-Port": "443",
        "X-Forwarded-Proto": "https",
        "X-Real-Ip": "192.168.3.213"
    },
    "myName": "bob"
}
```

### POST

A POST request expects a YAML request body that specifies the next upstream services that the app needs to send requests to, specified under `nextHops`. 

If a `serviceUrl` does not have `nextHops` specified it will send a GET request to this upstream app, if it does have `nextHops` specified, it will send a POST request, and it will send the part of the YAML request that are instructions for the next hops as a YAML request body.

This pattern will repeat itself according to how you specify the initial request.

**Request**

```
Content-Type = application/x-yaml
```

```
nextHops:
- serviceUrl: http://mesh-crawler-alice_apps-1-default_svc_8080.mesh
  nextHops:
  - serviceUrl: http://mesh-crawler-charlie_apps-2-default_svc_8080.mesh
  - serviceUrl: http://mesh-crawler-grace_apps-1-default_svc_8080.mesh
```

**Response**

```
{
    "myIncomingRequestheaders": {
        "Accept": "*/*",
        "Accept-Encoding": "gzip, deflate, br",
        "Connection": "keep-alive",
        "Content-Length": "233",
        "Content-Type": "text/plain",
        "Host": "mesh-crawler-bob_apps-1-default_svc_8080.mesh",
        "Postman-Token": "d5c8fcb4-0764-4636-997c-a0292574ada7",
        "User-Agent": "PostmanRuntime/7.29.0",
        "X-Forwarded-For": "192.168.3.213",
        "X-Forwarded-Host": "bob.mesh-zone-a.k8s.mschnkvld.lab",
        "X-Forwarded-Path": "/",
        "X-Forwarded-Port": "443",
        "X-Forwarded-Proto": "https",
        "X-Real-Ip": "192.168.3.213"
    },
    "myName": "bob",
    "myUpstreamResponses": [
        {
            "[POST] - http://mesh-crawler-alice_apps-1-default_svc_8080.mesh": {
                "myIncomingRequestheaders": {
                    "Accept": "*/*",
                    "Accept-Encoding": "gzip, deflate",
                    "Connection": "keep-alive",
                    "Content-Length": "150",
                    "Content-Type": "text/plain",
                    "Host": "mesh-crawler-alice_apps-1-default_svc_8080.mesh",
                    "Mesh-Crawler-Requester": "bob",
                    "User-Agent": "python-requests/2.28.0"
                },
                "myName": "alice",
                "myUpstreamResponses": [
                    {
                        "[GET] - http://mesh-crawler-charlie_apps-2-default_svc_8080.mesh": {
                            "myIncomingRequestheaders": {
                                "Accept": "*/*",
                                "Accept-Encoding": "gzip, deflate",
                                "Connection": "keep-alive",
                                "Content-Type": "text/plain",
                                "Host": "mesh-crawler-charlie_apps-2-default_svc_8080.mesh",
                                "Mesh-Crawler-Requester": "alice",
                                "User-Agent": "python-requests/2.28.0"
                            },
                            "myName": "charlie"
                        }
                    },
                    {
                        "[GET] - http://mesh-crawler-grace_apps-1-default_svc_8080.mesh": {
                            "myIncomingRequestheaders": {
                                "Accept": "*/*",
                                "Accept-Encoding": "gzip, deflate",
                                "Connection": "keep-alive",
                                "Content-Type": "text/plain",
                                "Host": "mesh-crawler-grace_apps-1-default_svc_8080.mesh",
                                "Mesh-Crawler-Requester": "alice",
                                "User-Agent": "python-requests/2.28.0"
                            },
                            "myName": "grace"
                        }
                    }
                ]
            }
        }
    ]
}
```

## Deployment

### Docker Build
I am using a private Docker registry to push the docker image to.
```
docker image build -t mesh-crawler:1.0.0 .
docker image tag mesh-crawler:1.0.0 <Docker Registry>/mesh-crawler:1.0.0
docker image push <Docker Registry>/mesh-crawler:1.0.0
```

### Kubernetes Deploy

Make sure to update the values between the curly brackets.

**Create an annotated namespace to make the app instance be part of a mesh.**
```
---
apiVersion: v1
kind: Namespace
metadata: 
  name: {{ meshCrawler.namespace }}
  namespace: {{ meshCrawler.namespace }}
  annotations: 
    kuma.io/sidecar-injection: enabled
    kuma.io/mesh: {{ meshCrawler.meshName }}
```

**Create a Kubernetes deployment and service**
```
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mesh-crawler-{{ meshCrawler.name }}
  labels:
    app: mesh-crawler-{{ meshCrawler.name }}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mesh-crawler-{{ meshCrawler.name }}
  template:
    metadata:
      labels:
        app: mesh-crawler-{{ meshCrawler.name }}
    spec:
      containers:
      - name: mesh-crawler
        image: docker-read-nexus.mschnkvld.lab:443/mesh-crawler:1.0.3
        env:
        - name: app_name
          value: "{{ meshCrawler.name }}"
        - name: app_port
          value: "8080"
        ports:
        - name: http
          containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: mesh-crawler-{{ meshCrawler.name }}
  labels:
    app: mesh-crawler-{{ meshCrawler.name }}
spec:
  ports:
  - name: http
    port: 8080
    targetPort: 8080
    appProtocol: http
  selector:
    app: mesh-crawler-{{ meshCrawler.name }}
  type: ClusterIP
```


## Next steps / Things to add
- Tracing (Jaeger / Zipkin)
- Health endpoint
- Security capabilities (oAuth, OIDC)
- TCP endpoints
- Improved execption handling
- Anything else?