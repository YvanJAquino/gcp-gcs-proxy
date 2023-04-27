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
- Caching:  gCSP uses LRU caching by default.  Basic caching (random eviction) is also available; LFU caching is planned for a future release.

# Target User Journeys
| As a developer, I can't use authenticated browser downloads **(https://storage.cloud.google.com/*/*)** because my organization requires Data Access Audit Logging for Cloud Storage. |
| :-- |
| Data Access Audit Logging prevents developers from using authenticated browser downloads (access) for private/internal-only objects.  gCSP uses the  running service's attached service account to access Cloud Storage, side-stepping this issue. |

| As a developer, I need to access or publicly display objects that are either private or exist within a private bucket |
| :-- |
| You have objects within a bucket that are either inaccessible publicly or the bucket in question has uniform access control policy for internal access only.  gCSP uses the running service's attached service account to access Cloud Storage, side-stepping this issue. |

| As an administrator, I'd like to prevent external users from accessing GCS URLs directly to prevent cost runs related to accidental or malicious usage. |
| :-- |
| gCSP is a reverse-proxy for Cloud Storage; reverse proxies protect backend resources. Run as a standalone binary in the background (as a background process) of the same host that's serving your web application.  You can provide access by configuring proxy options within your JS Framework that'll proxy requests back to the locally running  service. Alternatively, you can run gCSP separately, decoupling its capabilities from the local running host (allowing for independent scaling), and requring service-to-service authentication. gCSP-AP, A local authentication proxy that add's a token to outgoing requests, makes this easier to implement. *Please note that this will not stop egress usage; that is calculated on a per byte basis of what's being rendered OUTSIDE OF GCP!*|

# Deployment
Deployment will depend on how you want to deploy - and to what platform you want to deploy to.

## Default Deployment (Cloud Run)
The default deployment targets Cloud Run using Cloud Build.  Cloud Build will build the container, store it in the container registry (gcr.io/PROJECT_ID/gcp-gcs-proxy), and then deploy gCSP to Cloud Run with no CPU throttling and requiring authentication.  This deployment model is meant to be used in conjunction with gCSP-AP. 

```shell
git clone https://github.com/YvanJAquino/gcp-gcs-proxy.git
cd gcp-gcs-proxy
gcloud builds submit
```

## Manual Deployments - Building the container with DockerYou can manually build the container and store it in the Container Registry for usage on other compute platforms like Kubernetes Engine or Compute Engine by running COS (Container-Optimized OS).

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

## Manual Deployments - Using Docker to build binaries if Go isn't installed locally.
Docker is a powerful tool in the right hands.  If you don't have Go installed, you can wrap your shell scripts into a Dockerfile, mount a separate volume, and then copy the resulting binary to the mounted volume.

1. create a new Dockerfile:
```Dockerfile
FROM    golang:1.18-buster as builder
WORKDIR /app
COPY    . ./
RUN     go build -ldflags="-w -s" -o service ./cmd/auth-proxy
```

2. build the container:
```shell
docker build -t local.gcsp-ap.builder .
```

3. Run the container in interactive mode
```shell
docker run -it --rm -v $(pwd)/bin:/exports /bin/bash
```

4. Copy over the resulting binary to the mounted volume. 
```shell
cp /app/service /exports/gcsp-ap
exit
```

5. Check the /bin folder.
```shell
ls -al bin | grep "gcsp-ap"
```


## Manual Deployments -Building and running the gCSP Auth Proxy (gCSP-AP) from source
gCSP's Auth Proxy (gCSP-AP) adds an OIDC identity token generated from Compute Engine's metadata server process to outgoing requests to gCSP.  gCSP-AP is required when gCSP is deployed separately on a service that requires service-to-service authentication.  gCSP-AP designed to run as a background process in the same host that's serving your web application.  


```shell
git clone https://github.com/YvanJAquino/gcp-gcs-proxy.git
cd gcp-gcs-proxy
go build -ldflags="-w -s" -o gcsp-ap ./cmd/auth-proxy
```

gCSP-AP requires further configuration of the web application's server so that outbound requests from the web app pass through gCSP-AP instead.   

### Configuring gCSP-AP
gCSP-AP can be configured through runtime environment variables.  These MUST be provided for gCSP-AP to work properly:

| name | purpose | 
| :-- | :-- | 
| GCSP_PROXY_PORT | gCSP-AP's serving port |
| GCSP_TARGET_ADDR | gCSP's address | 

# Testing
Testing should be done incrementally, in steps, to ensure that each part of the system is operating as desired.

Local testing requires that Go 1.18 or above is installed.

## Default Deployment (gCSP on Cloud Run with gCSP-AP)
Begin by deploying using the default deployment.  

```shell
git clone https://github.com/YvanJAquino/gcp-gcs-proxy.git
cd gcp-gcs-proxy
gcloud builds submit
```

This creates a service using the default name `gcp-gcs-proxy` in the default region `us-central1`.  We provide a port via GCSP_PROXY_PORT and the the service's URL as target address for gCSP-AP.  

```shell
export GCSP_PROXY_PORT=10274
export SERVICE=gcp-gcs-proxy && \
export REGION=us-central1 && \
export GCSP_TARGET_ADDR=$(gcloud run services describe --region=$REGION --format="value(status.address.url)" $SERVICE)

echo "gCSP-AP upstream is: $GCSP_TARGET_ADDR"
```

If you don't see your Cloud Run service's URL, consider doing a manual deployment until you find the problem.

With the local environment configured, run gCSP-AP:

```shell
go run cmd/auth-proxy/main.go
```

```shell
### OUTPUT ###
2023/04/26 13:26:38 HOST:  - PORT: 10274 - TARGET: https://gcp-gcs-proxy-*.a.run.app
2023/04/26 TargetURL: https://gcp-gcs-proxy-*.a.run.app
2023/04/26 Serving traffic from :10274
```

From another shell, use curl to query gCSP-AP (which then queries gCSP running on Cloud Run!)

```shell
curl localhost:10274/storage/v1/b
```
```shell
### OUTPUT ###
[
    "holy-diver-297719",
    "holy-diver-297719-input",
    "holy-diver-297719-labs",
    "holy-diver-297719-output",
    "holy-diver-297719-private",
    "holy-diver-297719-public",
    "holy-diver-297719-reports",
    "holy-diver-297719.appspot.com",
    "holy-diver-297719_cloudbuild",
]
```

With tests out of the way, you can build gCSP-AP for continued local usage.  For deployment, it is recommended to build gCSP-AP in a separate build step, insert into your final container image, and then run gCSP-AP as a background process.  

Building locally:

```shell
go build -ldflags="-w -s" -o gcsp-ap ./cmd/auth-proxy
```

# Integrating gCSP-AP with Modern Javascript Frameworks.

## Frameworks that use Vite 
 
 Vite configuration documentation: https://vitejs.dev/config/

Use Vite's proxy configuration option to route 'relative' traffic back to gCSP's running port.  In this example, traffic destined for /storage/v1/b is routed to http://localhost:1337/ since gCSP-AP is running on 1337.  

```typescript
// vite.config.ts for SolidJS
export default defineConfig({
  plugins: [solidPlugin()],
  server: {
    proxy: {
      '/storage/v1/b': 'http://localhost:1337/'
    },
    port: 3000,
    host: '0.0.0.0',
  },
  build: {
    target: 'esnext',
  },
});
```

Render storage images within your application as if they were locally available objects within `<img>` tags.

```typescript
import { createEffect, createSignal } from "solid-js";

async function listBuckets() {
	const result = await fetch('storage/v1/b');
	return await result.text();
}

export default function Index() {
	const [buckets, setBuckets] = createSignal('');
	const [image, setImage] = createSignal('');

	createEffect(() => {
		listBuckets()
			.then(text => setBuckets(text));
	})

	return (<>
		<div>

			<h1>Hello World!</h1>
			<pre>
				{buckets()}			
			</pre>
			<img 
				src="storage/v1/b/super-mario-private/o/mario-is-awesome?alt=media"
				width={500}
			/>
		</div>
	</>)
}
```

## Frameworks that use ExpressJS

## Next.js
Next.js documentation: https://nextjs.org/docs/api-reference/next.config.js/rewrites


# Planned features
- Advanced caching eviction strategies.  In particular:
    - LFU (Least Frequencly Used) eviction strategy
- Traditional configuration options through environment variables.  
- Usage examples that align to each of the Target User Journeys
- Documentation.  