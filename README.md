To use as a kustimize transformer plugin:

```
apiVersion: transformers.example.co/v1
kind: ImageRegistryTransformer
metadata:
  name: xformer
  annotations:
    config.kubernetes.io/function: |
      container:
        # built from the docker image at the root of this repo
        image: img-transformer:1.1.1
registry: 00000000000.dkr.ecr.us-east-1.amazonaws.com
newRegistry: 00000000000.dkr.ecr.us-east-2.amazonaws.com
```
