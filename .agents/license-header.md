# License Header

## Guideline

- Start every new code file, including test files, with the license header below; report an existing code file that lacks it instead of adding it, unless the task covers it.
- The header comes first in the code file, followed by a blank line; only a line a tool requires at the very top (a shebang, an encoding line) stays above it.
- The header text below is plaintext; when writing it into a code file, prefix every line with the file language's line-comment token:
  - `go`: `//`
- Formats without line comments (e.g. Markdown, JSON) don't carry the header, and neither do data formats that have them (e.g. YAML, TOML).
<!-- Project-specific / Guideline -->

## Full header

The default header for new code file is:

```
Copyright 2025 The Viction Authors
(modifications)
This file is part of the go-ethereum library.

The go-ethereum library is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

The go-ethereum library is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.
```

<!-- Project-specific / Guideline -->

If editing existing code file, modify header follow this form:
```
Copyright 2017 The go-ethereum Authors
(original work)
Copyright 2025 The Viction Authors
(modifications)
This file is part of the go-ethereum library.
```

## Project-specific

<!-- Project-specific -->
