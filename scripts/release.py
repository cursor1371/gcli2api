#!/usr/bin/env python3
"""
Release automation script for gcli2api project.
Handles building cross-platform binaries and uploading releases to GitHub.
"""

import os
import subprocess
import sys
import zipfile
import tarfile
import hashlib
from datetime import datetime
from pathlib import Path
from typing import List, Optional, Dict

import click


def run_command(
    cmd: List[str],
    cwd: Optional[Path] = None,
    check: bool = True,
    env: Optional[Dict[str, str]] = None,
    capture_output: bool = False,
) -> subprocess.CompletedProcess:
    """Run a command and return the result."""
    click.echo(f"Running: {' '.join(cmd)}")
    return subprocess.run(
        cmd,
        cwd=cwd,
        check=check,
        capture_output=capture_output,
        env=env,
        text=capture_output,
    )


def calculate_sha256(file_path: Path) -> str:
    """Calculate SHA256 hash of a file."""
    sha256_hash = hashlib.sha256()
    with open(file_path, "rb") as f:
        for byte_block in iter(lambda: f.read(4096), b""):
            sha256_hash.update(byte_block)
    return sha256_hash.hexdigest()


@click.group()
def cli():
    """Release automation for gcli2api project."""
    pass


@cli.command("build-release")
@click.option("--app-name", default="gcli2api", help="Application name for binaries")
@click.option("--dist-dir", default="dist", help="Distribution directory")
def build_release(app_name: str, dist_dir: str):
    """Build cross-platform binaries for release."""
    click.echo("Building cross-platform binaries...")

    dist_path = Path(dist_dir)
    dist_path.mkdir(exist_ok=True)

    platforms = [
        ("linux", "amd64"),
        ("linux", "arm64"),
        ("darwin", "arm64"),
        ("windows", "amd64"),
    ]

    for goos, goarch in platforms:
        ext = ".exe" if goos == "windows" else ""
        archive_ext = "zip" if goos == "windows" else "tar.gz"

        bin_name = f"{app_name}_{goos}_{goarch}{ext}"
        bin_path = dist_path / bin_name

        click.echo(f"Building {bin_path}")

        # Build the binary
        env = os.environ.copy()
        env.update(
            {
                "CGO_ENABLED": "0",
                "GOOS": goos,
                "GOARCH": goarch,
            }
        )

        run_command(
            ["go", "build", "-trimpath", "-ldflags", "-s -w", "-o", str(bin_path), "."],
            env=env,
        )

        # Create archive
        pkg_name = f"{app_name}_{goos}_{goarch}.{archive_ext}"
        pkg_path = dist_path / pkg_name

        if archive_ext == "zip":
            with zipfile.ZipFile(pkg_path, "w", zipfile.ZIP_DEFLATED) as zf:
                zf.write(bin_path, bin_name)
        else:
            with tarfile.open(pkg_path, "w:gz") as tf:
                tf.add(bin_path, bin_name)

        # Remove the binary after archiving
        bin_path.unlink()
        click.echo(f"Created {pkg_path}")

    # Generate checksums
    click.echo("Generating checksums...")
    checksum_path = dist_path / "SHA256SUMS.txt"

    with open(checksum_path, "w") as f:
        for file_path in sorted(dist_path.glob("*")):
            if file_path.name != "SHA256SUMS.txt" and file_path.is_file():
                sha256 = calculate_sha256(file_path)
                f.write(f"{sha256}  {file_path.name}\n")

    click.echo(f"Checksums written to {checksum_path}")
    click.echo("Build completed successfully!")


@cli.command("upload-release")
@click.option("--dist-dir", default="dist", help="Distribution directory")
@click.option(
    "--github-token", envvar="GITHUB_TOKEN", required=True, help="GitHub token"
)
@click.option(
    "--github-sha", envvar="GITHUB_SHA", required=True, help="GitHub commit SHA"
)
def upload_release(dist_dir: str, github_token: str, github_sha: str):
    """Create or update GitHub release and upload assets."""
    click.echo("Creating/updating GitHub release...")

    dist_path = Path(dist_dir)
    if not dist_path.exists():
        click.echo(
            f"Error: Distribution directory {dist_path} does not exist", err=True
        )
        sys.exit(1)

    sha7 = github_sha[:7]
    timestamp = datetime.utcnow().strftime("%Y%m%d-%H%M%S")
    tag = f"nightly-{timestamp}-{sha7}"
    title = f"nightly-{timestamp}-{sha7}"

    # Check if release exists
    release_exists = False
    try:
        run_command(["gh", "release", "view", tag], check=True, capture_output=True)
        release_exists = True
    except subprocess.CalledProcessError:
        pass

    if release_exists:
        click.echo(f"Release {tag} exists; will update assets.")
    else:
        # Create the release
        click.echo(f"Creating release {tag}...")

        # Generate changelog
        notes = f"Automated nightly build for commit {github_sha}"
        try:
            # Get latest release tag from GitHub
            result = run_command(
                [
                    "gh",
                    "release",
                    "list",
                    "--limit",
                    "1",
                    "--json",
                    "tagName",
                    "-q",
                    ".[0].tagName",
                ],
                capture_output=True,
                check=False,
            )
            last_tag = (
                result.stdout.strip()
                if result.returncode == 0 and result.stdout.strip()
                else None
            )

            if last_tag:
                click.echo(f"Found last release tag: {last_tag}")
                log_cmd = [
                    "git",
                    "log",
                    f"{last_tag}..HEAD",
                    "--oneline",
                    "--pretty=format:* %h %s",
                ]
                log_result = run_command(log_cmd, capture_output=True, check=False)
                if log_result.returncode == 0 and log_result.stdout.strip():
                    notes = f"## Changelog\n\n{log_result.stdout.strip()}"
                else:
                    click.echo(
                        f"Could not get git log between {last_tag} and HEAD. Using default notes.",
                        err=True,
                    )
            else:
                click.echo("No previous release found. Using default notes.")

        except Exception as e:
            click.echo(f"Could not generate changelog: {e}", err=True)

        run_command(
            [
                "gh",
                "release",
                "create",
                tag,
                "--title",
                title,
                "--notes",
                notes,
                "--prerelease",
            ]
        )

    # Upload/overwrite assets
    click.echo("Uploading assets...")
    assets = list(dist_path.glob("*"))
    if not assets:
        click.echo("Warning: No assets found to upload", err=True)
        return

    asset_paths = [str(asset) for asset in assets if asset.is_file()]
    run_command(["gh", "release", "upload", tag, "--clobber"] + asset_paths)

    click.echo(f"Successfully uploaded {len(asset_paths)} assets to release {tag}")


if __name__ == "__main__":
    cli()
