---
category: Basic
group: Application
title: Environment Variables
weight: 1
---

# Environment Variables

## Overview

Environment variables are a set of dynamic named values that can affect the way running processes will behave on a computer. They are part of the environment in which a process runs.

Configure extra environment variables:

```yaml
kind: Application
# ...
spec:
	# ...
	extraEnv:
		name: FOO
		value: bar
```
