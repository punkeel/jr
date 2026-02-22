# jr

A CLI tool for managing long-running jobs via systemd user units.

## Overview

`jr` (job run) allows you to start, monitor, and manage long-running jobs that survive SSH disconnects. Jobs run as systemd user units and can be monitored from any session.

## Installation

```bash
make install
```

## Usage

```bash
jr start <command> [args...]  # Start a new job
jr list                       # List all jobs
jr status <id>                # Show job status
jr logs <id>                  # View job logs
jr stop <id>                  # Stop a job
jr rm <id>                    # Remove a job
jr prune                      # Remove stopped jobs
jr doctor                     # Check system health
```

## Requirements

- Go 1.25+
- systemd (with user units support)
