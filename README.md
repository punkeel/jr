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
jr run [flags] -- <command> [args...]  # Run a new job (alias: start)
  jr run -a -- <command>                # Run and attach to output (Ctrl+C detaches)
jr list                                # List all jobs
jr status <id>                         # Show job status
jr logs <id>                           # View job logs
  jr logs --raw <id>                    # View logs without timestamp/hostname prefix
jr stop <id>                           # Stop a job
jr rm <id>                             # Remove a job
jr prune                               # Remove old jobs
jr doctor                              # Check system health (with colors!)
```

## Requirements

- Go 1.25+
- systemd (with user units support)
