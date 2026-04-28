# frozen_string_literal: true
#
# SDP Toolkit — Homebrew Formula (DRY-RUN / REHEARSAL)
#
# This is a rehearsal formula for F150-08. It is NOT published to a tap.
# The formula installs only the `sdp` CLI binary from SDP Toolkit.
#
# Lab-only binaries (sdp-control, sdp-dispatch, sdp-up, sdp-harness, sdp-a2a,
# sdp-strataudit, etc.) and research/benchmark binaries are intentionally
# excluded. See docs/reference/maturity-matrix.md for full classification.
#
# Dry-run usage:
#   brew install --formula ./formula/sdp.rb
#   brew test ./formula/sdp.rb
#   brew uninstall sdp
#
# Actual tap publishing is out of scope until explicit approval.
#
class Sdp < Formula
  desc "SDP Toolkit — governed AI software delivery harness CLI"
  homepage "https://github.com/fall-out-bug/sdp_lab"
  # Version and SHA256 are placeholders for dry-run; the dry-run script
  # updates them automatically from the local build.
  version "0.0.0-dryrun"
  license "MIT"

  # Source: use a GitHub archive tarball for the tagged release.
  # For dry-run, the script builds from local source and rewrites these.
  url "https://github.com/fall-out-bug/sdp_lab/archive/refs/heads/main.tar.gz"
  # sha256 is set by the dry-run script; placeholder here
  sha256 "PLACEHOLDER_WILL_BE_OVERWRITTEN_BY_DRY_RUN"

  depends_on "go" => :build

  def install
    # Build the sdp binary from source
    # The ldflags inject version, commit, and date at build time
    commit = Utils.popen_read("git", "rev-parse", "--short", "HEAD").strip
    build_date = Time.now.utc.strftime("%Y-%m-%dT%H:%M:%SZ")

    system "go", "build", "-tags", "sqlite_fts5",
           "-ldflags",
           "-s -w -X main.version=#{version} -X main.commit=#{commit} -X main.date=#{build_date}",
           "-o", bin/"sdp",
           "./cmd/sdp"
  end

  test do
    # Test 1: sdp --help exits successfully and shows usage
    output = shell_output("#{bin}/sdp 2>&1", 2)
    assert_match "usage: sdp <command>", output

    # Test 2: scout --help shows a read-only subcommand usage
    scout_output = shell_output("#{bin}/sdp scout --help 2>&1")
    assert_match "output format", scout_output
  end
end
