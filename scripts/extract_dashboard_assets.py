#!/usr/bin/env python3
import re
import sys


ASSET_PATTERN = re.compile(r"/_app/immutable/[A-Za-z0-9._/-]+")


def extract_asset_paths(document: str) -> list[str]:
    return sorted(set(ASSET_PATTERN.findall(document)))


def main() -> None:
    for asset in extract_asset_paths(sys.stdin.read()):
        print(asset)


if __name__ == "__main__":
    main()
