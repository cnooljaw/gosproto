"""校验历史档案的整体 SHA256、文件集合及逐文件内容，不解压或执行归档。"""
import argparse
import hashlib
import json
from pathlib import Path
import tarfile


def sha256(stream):
    digest = hashlib.sha256()
    for chunk in iter(lambda: stream.read(1024 * 1024), b""):
        digest.update(chunk)
    return digest.hexdigest()


def verify(archive, index):
    with archive.open("rb") as stream:
        if sha256(stream) != index["archive"]["sha256"]:
            raise ValueError("archive SHA256 mismatch")
    if archive.stat().st_size != index["archive"]["bytes"]:
        raise ValueError("archive size mismatch")
    manifest_path = index["manifestPath"]
    prefix = manifest_path.removesuffix("ARCHIVE-MANIFEST.json")
    with tarfile.open(archive, "r|gz") as tar:
        first = tar.next()
        if first is None or not first.isfile() or first.name != manifest_path:
            raise ValueError("archive manifest missing or misplaced")
        manifest = json.load(tar.extractfile(first))
        if manifest["sourceCommit"] != index["sourceCommit"]:
            raise ValueError("source commit mismatch")
        entries = manifest["files"]
        expected = {prefix + item["path"]: item for item in entries}
        if len(expected) != len(entries) or len(entries) != index["files"]:
            raise ValueError("manifest file count mismatch")
        total = 0
        while (member := tar.next()) is not None:
            item = expected.pop(member.name, None)
            if item is None or not member.isfile():
                raise ValueError(f"unexpected archive member: {member.name}")
            if member.size != item["bytes"] or sha256(tar.extractfile(member)) != item["sha256"]:
                raise ValueError(f"file mismatch: {member.name}")
            total += member.size
        if expected or total != index["uncompressedBytes"]:
            raise ValueError("archive content is incomplete")
    print(f"Verified {len(entries)} files, {total} bytes; source {index['sourceCommit']}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("archive", type=Path)
    parser.add_argument("--index", type=Path, default=Path(__file__).resolve().parents[2] / "docs/benchmarks/ARCHIVE.json")
    args = parser.parse_args()
    try:
        verify(args.archive, json.loads(args.index.read_text()))
    except (OSError, ValueError, KeyError, tarfile.TarError) as error:
        parser.exit(1, f"Archive verification failed: {error}\n")
