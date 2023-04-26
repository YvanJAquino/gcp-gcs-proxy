# Cloud Storage Proxy (gCSP)

Cloud Storage Proxy (gCSP) is a reverse-proxy for Google Cloud Storage, gGCP's fully-managed object storage service.  gCSP's API endpoints map to some of Cloud Storage's JSON API endpoints to provide extra utility such as listing buckets, listing objects, and getting object metadata in addition to it's core functionality: getting object data (media) from GCS.  

# Features
- Simple: gCSP is easy for administrators to set-up and easy for developers to use.  Under the hood, gCSP compiles to a single binary with no required operating system dependencies.  
- Stateless:  gCSP happily lives in ephemeral containers. 
- Support for basic, key operations: gCSP provides endpoints that map to GCS' JSON API for listing buckets, listing objects in a bucket, and getting object-level metadata: 

    | Endpoint | Purpose |
    | :-- | :-- |
    | /storage/v1/b | listing buckets |
    | /storage/v1/b/BUCKET/o | listing objects in BUCKET |
    | /storage/v1/b/BUCKET/o/OBJECT | listing object metadata |
    | /storage/v1/b/BUCKET/o/OBJECT?alt=media | getting object data |

- Service agnostic: Run gCSP anywhere containers are accepted like Cloud Run, GKE, GCE, or App Engine Flex.  Alternatively, build from source and run as a background process on the same host as your web application.
- Caching:  gCSP uses LRU caching by default.  Basic caching (random eviction) is also available; LFU caching is planned for a future release .  

# Deployment

Deployment will depend on how you want to deploy - and to what platform you want to deploy to.

## Default Deployment (Cloud Run)

The default deployment targets Cloud Run using Cloud Build.  Cloud Build will build the container, store it in the container registry (gcr.io/PROJECT_ID/gcp-gcs-proxy), and then deploy gCSP to Cloud Run with no CPU throttling and requiring authentication.  This deployment model is meant to be used in conjunction with gCSP-AP. 

```shell
git clone https://github.com/YvanJAquino/gcp-gcs-proxy.git
cd gcp-gcs-proxy
gcloud builds submit
```

## Manual Deployments - Building the container with Docker

You can manually build the container and store it in the Container Registry for usage on other compute platforms like Kubernetes Engine or Compute Engine by running COS (Container-Optimized OS).

```
git clone https://github.com/YvanJAquino/gcp-gcs-proxy.git
cd gcp-gcs-proxy
docker build -t gcr.io/PROJECT_ID/gcp-gcs-proxy . 
docker push gcr.io/PROJECT_ID/gcp-gcs-proxy
```

## Manual Deployments - Building the proxy from source

You can build the binary locally if you have Go 1.18 or greater installed.  This deployment method is ideal if you're going to run gCSP as background process on the same host that's serving your web application.  

When containerizing your web application, you can use this snippet below in a separate build step to build the binary, copy it to the final execution step, and then run it in the background.  

```
git clone https://github.com/YvanJAquino/gcp-gcs-proxy.git
cd gcp-gcs-proxy
go build -ldflags="-w -s" -o gcsp-proxy ./cmd/gcs-proxy
```

## Building and running the gCSP Auth Proxy (gCSP-AP)

gCSP's Auth Proxy (gCSP-AP) adds an OIDC identity token generated from Compute Engine's metadata server process to outgoing requests to gCSP.  gCSP-AP is required when gCSP is deployed separately on a service that requires service-to-service authentication.  gCSP-AP designed to run as a background process in the same host that's serving your web application.  


```shell
git clone https://github.com/YvanJAquino/gcp-gcs-proxy.git
cd gcp-gcs-proxy
go build -ldflags="-w -s" -o auth-proxy ./cmd/auth-proxy
```

gCSP-AP requires further configuration of the web application's server so that outbound requests from the web app pass through gCSP-AP instead.   

### Configuring gCSP-AP
gCSP-AP can be configured through runtime environment variables.  These MUST be provided for gCSP-AP to work properly:

| name | purpose | 
| :-- | :-- | 
| GCSP_PROXY_PORT | gCSP-AP's serving port |
| GCSP_TARGET_ADDR | gCSP's address | 

# Target User Journeys

| As a developer, I can't use authenticated browser downloads **(https://storage.cloud.google.com/\*/\*)** because my organization requires Data Access Audit Logging for Cloud Storage. |
| :-- |
| Data Access Audit Logging prevents developers from using authenticated browser downloads (access) for private/internal-only objects.  gCSP uses the  running service's attached service account to access Cloud Storage, side-stepping this issue. |

| As a developer, I need to access or publicly display objects that are either private or exist within a private bucket |
| :-- |
| You have objects within a bucket that are either inaccessible publicly or the bucket in question has uniform access control policy for internal access only.  gCSP uses the running service's attached service account to access Cloud Storage, side-stepping this issue.  This has the added benefit of protecting the object's URL from direct abuse. |

| As an administrator, I'd like to prevent external users from accessing GCS URLs directly to prevent cost runs related to accidental or malicious usage. |
| :-- |
| gCSP is a reverse-proxy for Cloud Storage; it can address this use-case in various ways. Run as a standalone binary in the background (as a background process) of the same host that's serving your web application.  You can provide access by configuring proxy options within your JS Framework that'll proxy requests back to the locally running  service. Alternatively, you can run gCSP separately, decoupling its capabilities from the local running host (allowing for independent scaling), and requring service-to-service authentication. gCSP-AP, A local authentication proxy that add's a token to outgoing requests, makes this easier to implement. |

# Planned features
- Advanced caching eviction strategies.  In particular:
    - LFU (Least Frequencly Used) eviction strategy
- Traditional configuration options through environment variables.  
- Usage examples that align to each of the Target User Journeys
- Documentation.  