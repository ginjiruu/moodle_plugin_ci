# Plugin CI

# Purpose

Provide a standardized CI configuration for running the moodle ci suite against plugins

# Warning

The gitlab-ci file included is currently quite heavy in the amount of jobs it will try to run. You may want to reduce the size of the "matrix" options in the file to only run agains the php versions and databases you want to test against.

## Local Execution

### Requirements
- https://taskfile.dev/installation/
- https://docs.dagger.io/install/

### Running

1. Install the requirements
2. Copy the Taskfile.yaml file to your local plugin repository  
(Alternatively run the dagger commands directly)

After installing task and dagger you should now be able to use the "task" command to execute all pipeline jobs at once.  
If you would like to exececute a single task then simply run the "task" command followed by the operation.

```bash
task lint
task cs
task save
task ...
```

All available tasks can be viewed through the "Taskfile.yaml" file

## Pipeline execution

To run these tasks automatically when pushing to a git repo add the .gitlab-ci.dist.yml file to your project repository and rename it to .gitlab-ci.yml
