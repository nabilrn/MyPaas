from pathlib import Path

path = Path("backend/internal/container/docker.go")
text = path.read_text()
old = '''\t\t\t\t\t\targ := fmt.Sprint(test[i])
\t\t\t\t\t\tif strings.ContainsAny(arg, " \\t\\\"'") {
\t\t\t\t\t\t\tbuilder.WriteString("'")
\t\t\t\t\t\t\tbuilder.WriteString(strings.ReplaceAll(arg, "'", "'\\\\''"))
\t\t\t\t\t\t\tbuilder.WriteString("'")
\t\t\t\t\t\t} else {
\t\t\t\t\t\t\tbuilder.WriteString(arg)
\t\t\t\t\t\t}
'''
new = '''\t\t\t\t\t\targ := fmt.Sprint(test[i])
\t\t\t\t\t\tbuilder.WriteString("'")
\t\t\t\t\t\tbuilder.WriteString(strings.ReplaceAll(arg, "'", "'\\\\''"))
\t\t\t\t\t\tbuilder.WriteString("'")
'''
if old not in text:
    raise SystemExit("expected healthcheck conversion block not found")
path.write_text(text.replace(old, new, 1))
