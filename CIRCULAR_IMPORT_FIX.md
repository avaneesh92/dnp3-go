# Circular Import Fix

## Problem

The original implementation had a circular import:
- `pkg/dnp3` imported `pkg/master`
- `pkg/master` imported `pkg/dnp3` (for config types)

## Solution

**Separated configuration types from public API:**

1. Moved config types to each implementation package:
   - `pkg/master/config.go` - MasterConfig, MasterCallbacks, SOEHandler, etc.
   - `pkg/outstation/config.go` - OutstationConfig, OutstationCallbacks, etc.

2. Public API (`pkg/dnp3`) maintains its own config types and wraps internal types:
   - `dnp3.MasterConfig` → converts to → `master.MasterConfig`
   - `dnp3.MasterCallbacks` → wraps → `master.MasterCallbacks`
   - Same pattern for Outstation

3. Factory files handle conversion:
   - `pkg/dnp3/master_factory.go` - Converts config and wraps callbacks
   - `pkg/dnp3/outstation_factory.go` - Converts config and wraps callbacks

## Result

No circular imports! Each package is independent:
- `pkg/master` - Has own types, no dnp3 import
- `pkg/outstation` - Has own types, no dnp3 import
- `pkg/dnp3` - Imports master/outstation, provides public API

## Additional Fixes

After initial separation, found two more issues:

1. **pkg/outstation/outstation.go line 180**: Fixed malformed `select` statement that had `if` inside `select` block
   - Changed to proper `case o.updateChan <- req:` syntax

2. **pkg/outstation/update_builder.go**: Removed remaining `import "avaneesh/dnp3-go/pkg/dnp3"`
   - Changed `Build()` to return `*Updates` instead of `*dnp3.Updates`
   - Now properly creates Updates with internal data map

## Build Command

```bash
cd e:\go\dnp3-go
go build ./...
```

Should now compile without circular import errors!
