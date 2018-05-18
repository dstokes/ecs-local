# ecs-local

Run ECS task definitions locally.

## Usage

```shell
ecs-local [-hv] [--profile=aws_profile] [--region=aws_region] [-e key=value] [task_def] [command...]
```

## Install

### macOS

```shell
brew tap fullscreen/tap
brew install ecs-local
```

# Caveats

## Network Access

This utility
creates docker containers
on your local machine
based off of task definitions
in ECS.
As a result
the spawned task
may not have
the same network access
as the remote ECS task.

## Assuming Roles

`ecs-local` will attempt
to assume the role
of the specified task.
If it is unable to do so
it will fail silently
and
a warning message
will be printed
if the `verbose` flag
is passed.
