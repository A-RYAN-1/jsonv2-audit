# Security policy

## Supported versions

Only the latest commit on `main` receives fixes. There is no release yet.

## Reporting a vulnerability

Please do not open a public issue. Use GitHub's private vulnerability reporting on this repository (Security tab,
"Report a vulnerability"). Expect an acknowledgement within 7 days and a status update within 14.

## Scope

The analyzer reads Go source and type information and never runs the code it checks. A crash or hang on crafted source, and anything that makes it write outside its output, are in scope.

A wrong or missing finding is a bug, not a vulnerability: open an issue with the smallest package that shows it.
