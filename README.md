# Cloud Storage Proxy (gCSP)

Cloud Storage Proxy (gCSP) is a reverse-proxy for Google Cloud Storage, gGCP's fully-managed object storage service.  gCSP's API endpoints map to some of Cloud Storage's JSON API endpoints to provide extra utility such as listing buckets, listing objects, and getting object metadata in addition to it's core functionality: getting object data (media) from GCS.  

# Features
- Simple: gCSP is easy for administrators to set-up and easy for developers to use.  Under the hood, gCSP compiles to a single binary with no required operating system binaries.  
- Stateless:  gCSP happily lives in ephemeral containers. 
- Support for basic, key operations: gCSP provides endpoints that map to GCS' JSON API for listing buckets, listing objects in a bucket, and getting object-level metadata: 

    | Endpoint | Purpose |
    | :-- | :-- |
    | /storage/v1/b | listing buckets |
    | /storage/v1/b/BUCKET/o | listing objects in BUCKET |
    | /storage/v1/b/BUCKET/o/OBJECT | listing object metadata |
    | /storage/v1/b/BUCKET/o/OBJECT?alt=media | getting object data |

- Service agnostic: Run gCSP anywhere containers are accepted like Cloud Run, GKE, GCE, or App Engine Flex.  Alternatively, build from source and run as a background process on the same host as your web application.
- Caching:  gCSP uses basic caching by default.  Advanced caching options are planned for a future release (LRU and LFU caching).  

# Deployment

Deployment will depend on how you want to deploy - and to what platform you want to deploy to.

## Default Deployment (Cloud Run)

The default deployment targets Cloud Run using Cloud Build.  Cloud Build will build the container, store it in the container registry (gcr.io/PROJECT_ID/gcp-gcs-proxy), and then deploy gCSP to Cloud Run with no CPU throttling and requiring authentication.  

```shell
git clone https://github.com/YvanJAquino/gcp-gcs-proxy.git
cd gcp-gcs-proxy
gcloud builds submit
```

## Manual Deployments

You can manually build the container and store it in the container registry for usage on other platforms like Kubernetes Engine or Compute Engine by running COS (Container-Optimized OS).

```
git clone https://github.com/YvanJAquino/gcp-gcs-proxy.git
cd gcp-gcs-proxy/service
docker build -t gcr.io/PROJECT_ID/gcp-gcs-proxy . 
docker push gcr.io/PROJECT_ID/gcp-gcs-proxy
```

# Target User Journeys

| As a developer, I can't use authenticated browser downloads **(https://storage.cloud.google.com/\*/\*)** because my organization requires Data Access Audit Logging for Cloud Storage. |
| :-- |
| Data Access Audit Logging prevents developers from using authenticated browser downloads (access) for private/internal-only objects.  gCSP uses the  running service's attached service account to access Cloud Storage, side-stepping this issue. |

| As a developer, I need to access or publicly display objects that are either private or exist within a private bucket |
| :-- |
| You have objects within a bucket that are either inaccessible publicly or the bucket in question has uniform access control policy for internal access only.  gCSP uses the running service's attached service account to access Cloud Storage, side-stepping this issue.  This has the added benefit of protecting the object's URL from direct abuse. |

| As an administrator, I'd like to prevent external users from accessing GCS URLs directly to prevent cost runs related to accidental or malicious usage. |
| :-- |
| gCSP is a reverse-proxy for Cloud Storage; it can address this use-case in various ways.  gCSP can be run as a standalone binary in the background of the same host that's serving your web application.  You can provide access by configuring proxy options that proxy requests back to the locally running gCSP service.  Alternatively, you can run gCSP separately, decoupling its capabilities from the local running host, and require service-to-service authentication. A local authentication proxy (that add's a token to outgoing requests) for GCP compute services is planned for a later release. |

# Planned features
- Advanced caching eviction strategies.  In particular:
    - LRU (Least Recently Used) eviction strategy
    - LFU (Least Frequencly Used) eviction strategy
- gCSP Authentication Proxy:  A small, simple proxy that accepts requests from the web applications local host and adds an Authorization header to outgoing requests for   service-to-service authentication to a separately run gCSP instance.
- Traditional configuration options through environment variables.  
- Usage examples that align to each of the Target User Journeys
- Documentation.  