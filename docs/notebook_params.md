# Notebook parameters

| Parameter        | Description  |
| ------------- |-------------|
|name| resource name defined the openvino_notebook image tag visible in the JupyterHub|
|git_refs| branch or tag in the github repository to be used to build the docker image|
|auto_update_image| set to `true` to enable automatic image rebuild then the github reposiory is updated on the configured branch. New image tag gets the suffix with the date of the image refresh|
|reconcile_duration_multiplier| increase the duration between github status testing comparing to standard reconcile duration in the operator; default is once in 60 which checks for github updates every 1h|

***

Check also:
- [OpenVINO notebook integration with RHODS](./notebook_in_rhods.md)