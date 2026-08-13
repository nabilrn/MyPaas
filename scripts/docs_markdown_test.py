import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
BOX_DRAWING = re.compile(r"[┌┐└┘├┤┬┴┼│─]")
ASCII_ARROW = re.compile(r"(?:<-{2,}|-{3,}>)")
FENCE = re.compile(r"```([^\n]*)\n(.*?)```", re.S)


class MarkdownDiagramTest(unittest.TestCase):
    def test_markdown_diagrams_use_rendered_mermaid(self) -> None:
        violations: list[str] = []
        for path in ROOT.rglob("*.md"):
            if ".git" in path.parts:
                continue
            text = path.read_text(encoding="utf-8")
            for match in FENCE.finditer(text):
                language = match.group(1).strip().lower()
                body = match.group(2)
                if language == "mermaid":
                    continue
                diagram_like = bool(BOX_DRAWING.search(body) or ASCII_ARROW.search(body))
                diagram_like = diagram_like or ("↓" in body and len([line for line in body.splitlines() if line.strip()]) >= 3)
                if diagram_like:
                    line = text.count("\n", 0, match.start()) + 1
                    violations.append(f"{path.relative_to(ROOT)}:{line}")
        self.assertEqual([], violations, "ASCII-style Markdown diagrams found; use Mermaid: " + ", ".join(violations))


if __name__ == "__main__":
    unittest.main()
