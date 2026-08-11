import unittest

from log_path import fields_by_line, latency_stats, percentile


class LogPathBenchmarkTest(unittest.TestCase):
    def test_fields_by_line_normalizes_empty_and_crlf(self) -> None:
        self.assertEqual(fields_by_line("web\r\n\ndb\n"), ["web", "db"])

    def test_percentile_interpolates(self) -> None:
        values = [1.0, 2.0, 3.0, 4.0]
        self.assertEqual(percentile(values, 0.5), 2.5)
        self.assertAlmostEqual(percentile(values, 0.95), 3.85)

    def test_latency_stats_contains_required_summary(self) -> None:
        stats = latency_stats([1.0, 2.0, 3.0])
        self.assertEqual(stats["min_ms"], 1.0)
        self.assertEqual(stats["mean_ms"], 2.0)
        self.assertEqual(stats["p50_ms"], 2.0)
        self.assertEqual(stats["max_ms"], 3.0)


if __name__ == "__main__":
    unittest.main()
