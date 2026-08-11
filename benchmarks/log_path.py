#!/usr/bin/env python3
"""Measure the current MyPaaS Compose log collection command path.

This intentionally mirrors DockerCLI.ComposeLogsAll rather than inventing a
new collector. It is a baseline harness for deciding whether a future native
log collector is justified.
"""

from __future__ import annotations

import argparse
import json
import math
import resource
import statistics
import subprocess
import time
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path


@dataclass
class Sample:
    elapsed_ms: float
    spawns: int
    services: int
    lines: int


class Runner:
    def __init__(self, docker_bin: str = "docker") -> None:
        self.docker_bin = docker_bin
        self.spawns = 0

    def run(self, *args: str) -> str:
        self.spawns += 1
        result = subprocess.run(
            [self.docker_bin, *args],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )
        return result.stdout


def fields_by_line(value: str) -> list[str]:
    return [line.strip() for line in value.replace("\r\n", "\n").split("\n") if line.strip()]


def percentile(values: list[float], percentile_value: float) -> float:
    if not values:
        raise ValueError("values are required")
    ordered = sorted(values)
    if len(ordered) == 1:
        return ordered[0]
    rank = (len(ordered) - 1) * percentile_value
    low = math.floor(rank)
    high = math.ceil(rank)
    if low == high:
        return ordered[low]
    weight = rank - low
    return ordered[low] * (1 - weight) + ordered[high] * weight


def latency_stats(values: list[float]) -> dict[str, float]:
    return {
        "min_ms": min(values),
        "mean_ms": statistics.fmean(values),
        "p50_ms": percentile(values, 0.50),
        "p95_ms": percentile(values, 0.95),
        "p99_ms": percentile(values, 0.99),
        "max_ms": max(values),
    }


def collect_compose_logs(runner: Runner, project: str, tail: int) -> Sample:
    start_spawns = runner.spawns
    started = time.perf_counter_ns()

    container_ids = fields_by_line(
        runner.run("ps", "-aq", "--filter", f"label=com.docker.compose.project={project}")
    )
    if not container_ids:
        raise RuntimeError(f"no Compose containers found for project {project!r}")

    service_names = fields_by_line(
        runner.run(
            "inspect",
            "--format",
            '{{ index .Config.Labels "com.docker.compose.service" }}',
            *container_ids,
        )
    )
    services = sorted({name for name in service_names if name and name != "<no value>"})
    if not services:
        raise RuntimeError(f"no Compose service labels found for project {project!r}")

    line_count = 0
    live_services = 0
    for service in services:
        ids = fields_by_line(
            runner.run(
                "ps",
                "-aq",
                "--filter",
                f"label=com.docker.compose.project={project}",
                "--filter",
                f"label=com.docker.compose.service={service}",
            )
        )
        if not ids:
            continue
        logs = runner.run("logs", "--tail", str(tail), ids[0])
        line_count += len(fields_by_line(logs))
        live_services += 1

    elapsed_ms = (time.perf_counter_ns() - started) / 1_000_000
    return Sample(
        elapsed_ms=elapsed_ms,
        spawns=runner.spawns - start_spawns,
        services=live_services,
        lines=line_count,
    )


def child_cpu_seconds() -> float:
    usage = resource.getrusage(resource.RUSAGE_CHILDREN)
    return usage.ru_utime + usage.ru_stime


def benchmark(project: str, tail: int, warmup: int, iterations: int, docker_bin: str) -> dict[str, object]:
    if iterations <= 0:
        raise ValueError("iterations must be positive")
    if warmup < 0:
        raise ValueError("warmup must not be negative")
    if tail <= 0:
        raise ValueError("tail must be positive")

    runner = Runner(docker_bin)
    for _ in range(warmup):
        collect_compose_logs(runner, project, tail)

    before_cpu = child_cpu_seconds()
    samples = [collect_compose_logs(runner, project, tail) for _ in range(iterations)]
    child_cpu = child_cpu_seconds() - before_cpu

    latencies = [sample.elapsed_ms for sample in samples]
    measured_spawns = sum(sample.spawns for sample in samples)
    service_counts = sorted({sample.services for sample in samples})

    return {
        "recorded_at": datetime.now(timezone.utc).isoformat(),
        "project": project,
        "tail": tail,
        "warmup": warmup,
        "iterations": iterations,
        "latency": latency_stats(latencies),
        "process_spawns": measured_spawns,
        "process_spawns_per_sample": measured_spawns / iterations,
        "child_cpu_seconds": child_cpu,
        "service_counts_observed": service_counts,
        "log_lines_per_sample": [sample.lines for sample in samples],
        "method": "mirrors DockerCLI.ComposeLogsAll: project ps + batched service inspect + per-service ps/logs",
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--project", required=True, help="Compose project label/name, e.g. mypaas-demo")
    parser.add_argument("--tail", type=int, default=100)
    parser.add_argument("--warmup", type=int, default=10)
    parser.add_argument("--iterations", type=int, default=100)
    parser.add_argument("--docker-bin", default="docker")
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()

    result = benchmark(args.project, args.tail, args.warmup, args.iterations, args.docker_bin)
    encoded = json.dumps(result, indent=2, sort_keys=True) + "\n"
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(encoded, encoding="utf-8")
    print(encoded, end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
