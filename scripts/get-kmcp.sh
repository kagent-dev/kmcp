#!/usr/bin/env bash

# Copyright The KMCP Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# The install script is based off of the MIT-licensed script from glide,
# the package manager for Go: https://github.com/Masterminds/glide.sh/blob/master/get

: ${BINARY_NAME:="kmcp"}
: ${USE_SUDO:="true"}
: ${DEBUG:="false"}
: ${VERIFY_CHECKSUM:="false"}
: ${KMCP_INSTALL_DIR:="/usr/local/bin"}

HAS_CURL="$(type "curl" &> /dev/null && echo true || echo false)"
HAS_WGET="$(type "wget" &> /dev/null && echo true || echo false)"
HAS_OPENSSL="$(type "openssl" &> /dev/null && echo true || echo false)"
HAS_GPG="$(type "gpg" &> /dev/null && echo true || echo false)"
HAS_TAR="$(type "tar" &> /dev/null && echo true || echo false)"
HAS_JQ="$(type "jq" &> /dev/null && echo true || echo false)"

# initArch discovers the architecture for this system.
initArch() {
  ARCH=$(uname -m)
  case $ARCH in
    armv5*) ARCH="armv5";;
    armv6*) ARCH="armv6";;
    armv7*) ARCH="arm";;
    aarch64) ARCH="arm64";;
    x86) ARCH="386";;
    x86_64) ARCH="amd64";;
    i686) ARCH="386";;
    i386) ARCH="386";;
  esac
}

# initOS discovers the operating system for this system.
initOS() {
  OS=$(echo `uname`|tr '[:upper:]' '[:lower:]')

  case "$OS" in
    # Minimalist GNU for Windows
    mingw*|cygwin*) OS='windows';;
  esac
}

# runs the given command as root (detects if we are root already)
runAsRoot() {
  if [ $EUID -ne 0 -a "$USE_SUDO" = "true" ]; then
    sudo "${@}"
  else
    "${@}"
  fi
}

# verifySupported checks that the os/arch combination is supported for
# binary builds, as well whether or not necessary tools are present.
verifySupported() {
  local supported="darwin-amd64\ndarwin-arm64\nlinux-amd64\nlinux-arm64"
  if ! echo "${supported}" | grep -q "${OS}-${ARCH}"; then
    echo "No prebuilt binary for ${OS}-${ARCH}."
    echo "To build from source, go to https://github.com/kagent-dev/kmcp"
    exit 1
  fi

  if [ "${HAS_CURL}" != "true" ] && [ "${HAS_WGET}" != "true" ]; then
    echo "Either curl or wget is required"
    exit 1
  fi

  if [ "${VERIFY_CHECKSUM}" == "true" ] && [ "${HAS_OPENSSL}" != "true" ]; then
    echo "In order to verify checksum, openssl must first be installed."
    echo "Please install openssl or set VERIFY_CHECKSUM=false in your environment."
    exit 1
  fi

  if [ "${HAS_TAR}" != "true" ]; then
    echo "[ERROR] Could not find tar. It is required to extract the KMCP binary archive."
    exit 1
  fi

  if [ "${HAS_JQ}" != "true" ]; then
    echo "[ERROR] Could not find jq. It is required to parse the KMCP version."
    exit 1
  fi
}

# checkDesiredVersion checks if the desired version is available.
checkDesiredVersion() {
  if [ "x$DESIRED_VERSION" == "x" ]; then
    # Get tag from release URL
    local latest_release_url="https://api.github.com/repos/kagent-dev/kmcp/releases/latest"
    local latest_release_response=""
    if [ "${HAS_CURL}" == "true" ]; then
      latest_release_response=$( curl -L --silent --show-error --fail "$latest_release_url" 2>&1 || true )
    elif [ "${HAS_WGET}" == "true" ]; then
      latest_release_response=$( wget "$latest_release_url" -q -O - 2>&1 || true )
    fi
    TAG=$( echo "$latest_release_response" | jq -r .tag_name | grep '^v[0-9]' )
    if [ "x$TAG" == "x" ]; then
      printf "Could not retrieve the latest release tag information from %s: %s\n" "${latest_release_url}" "${latest_release_response}"
      exit 1
    fi
  else
    TAG=$DESIRED_VERSION
  fi
}

# checkKmcpInstalledVersion checks which version of KMCP is installed and
# if it needs to be changed.
checkKmcpInstalledVersion() {
  if [[ -f "${KMCP_INSTALL_DIR}/${BINARY_NAME}" ]]; then
    local version=$("${KMCP_INSTALL_DIR}/${BINARY_NAME}" --version 2>/dev/null | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' || echo "unknown")
    if [[ "$version" == "$TAG" ]]; then
      echo "KMCP ${version} is already ${DESIRED_VERSION:-latest}"
      return 0
    else
      echo "KMCP ${TAG} is available. Changing from version ${version}."
      return 1
    fi
  else
    return 1
  fi
}

# downloadFile downloads the latest binary package and also the checksum
# for that binary.
downloadFile() {
  KMCP_DIST="${BINARY_NAME}-${OS}-${ARCH}"
  DOWNLOAD_URL="https://github.com/kagent-dev/kmcp/releases/download/${TAG}/${KMCP_DIST}"
  CHECKSUM_URL="${DOWNLOAD_URL}.sha256"
  KMCP_TMP_ROOT="$(mktemp -dt kmcp-installer-XXXXXX)"
  KMCP_TMP_FILE="$KMCP_TMP_ROOT/$KMCP_DIST"
  KMCP_SUM_FILE="$KMCP_TMP_ROOT/$KMCP_DIST.sha256"
  echo "Downloading $DOWNLOAD_URL"
  if [ "${HAS_CURL}" == "true" ]; then
    curl -SsL "$CHECKSUM_URL" -o "$KMCP_SUM_FILE" 2>/dev/null || true
    curl -SsL "$DOWNLOAD_URL" -o "$KMCP_TMP_FILE"
  elif [ "${HAS_WGET}" == "true" ]; then
    wget -q -O "$KMCP_SUM_FILE" "$CHECKSUM_URL" 2>/dev/null || true
    wget -q -O "$KMCP_TMP_FILE" "$DOWNLOAD_URL"
  fi
}

# verifyFile verifies the SHA256 checksum of the binary package
# and the GPG signatures for both the package and checksum file
# (depending on settings in environment).
verifyFile() {
  if [ "${VERIFY_CHECKSUM}" == "true" ]; then
    verifyChecksum
  fi
}

# installFile installs the KMCP binary.
installFile() {
  echo "Preparing to install $BINARY_NAME into ${KMCP_INSTALL_DIR}"
  runAsRoot chmod +x "$KMCP_TMP_FILE"
  runAsRoot cp "$KMCP_TMP_FILE" "$KMCP_INSTALL_DIR/$BINARY_NAME"
  echo "$BINARY_NAME installed into $KMCP_INSTALL_DIR/$BINARY_NAME"
}

# verifyChecksum verifies the SHA256 checksum of the binary package.
verifyChecksum() {
  if [ -f "$KMCP_SUM_FILE" ]; then
    printf "Verifying checksum... "
    local sum=$(openssl sha256 -sha256 ${KMCP_TMP_FILE} | awk '{print $2}')
    local expected_sum=$(cat ${KMCP_SUM_FILE} | awk '{print $1}')
    if [ "$sum" != "$expected_sum" ]; then
      echo "SHA sum of ${KMCP_TMP_FILE} does not match. Aborting."
      exit 1
    fi
    echo "Done."
  else
    echo "Warning: Checksum file not found, skipping verification."
  fi
}

# fail_trap is executed if an error occurs.
fail_trap() {
  result=$?
  if [ "$result" != "0" ]; then
    if [[ -n "$INPUT_ARGUMENTS" ]]; then
      echo "Failed to install $BINARY_NAME with the arguments provided: $INPUT_ARGUMENTS"
      help
    else
      echo "Failed to install $BINARY_NAME"
    fi
    echo -e "\tFor support, go to https://github.com/kagent-dev/kmcp."
  fi
  cleanup
  exit $result
}

# testVersion tests the installed client to make sure it is working.
testVersion() {
  set +e
  KMCP="$(command -v $BINARY_NAME)"
  if [ "$?" = "1" ]; then
    echo "$BINARY_NAME not found. Is $KMCP_INSTALL_DIR on your "'$PATH?'
    exit 1
  fi
  set -e
}

# addToPath adds the installation directory to the user's PATH if it's not already there
addToPath() {
  local shell_rc=""
  local path_added=false
  
  # Determine which shell RC file to use
  if [ -n "$ZSH_VERSION" ]; then
    shell_rc="$HOME/.zshrc"
  elif [ -n "$BASH_VERSION" ]; then
    shell_rc="$HOME/.bashrc"
  else
    # Default to .bashrc if we can't determine the shell
    shell_rc="$HOME/.bashrc"
  fi
  
  # Check if the directory is already in PATH
  if [[ ":$PATH:" != *":$KMCP_INSTALL_DIR:"* ]]; then
    echo "Adding $KMCP_INSTALL_DIR to your PATH in $shell_rc"
    echo "" >> "$shell_rc"
    echo "# KMCP binary path" >> "$shell_rc"
    echo "export PATH=\"$KMCP_INSTALL_DIR:\$PATH\"" >> "$shell_rc"
    path_added=true
  fi
  
  if [ "$path_added" = true ]; then
    echo ""
    echo "PATH updated! Please run one of the following to reload your shell configuration:"
    echo "  source $shell_rc"
    echo "  or restart your terminal"
  fi
}

# help provides possible cli installation arguments
help () {
  echo "Accepted cli arguments are:"
  echo -e "\t[--help|-h ] ->> prints this help"
  echo -e "\t[--version|-v <desired_version>] . When not defined it fetches the latest release tag from the KMCP GitHub repository"
  echo -e "\te.g. --version v0.1.0 or -v canary"
  echo -e "\t[--no-sudo]  ->> install without sudo"
}

# cleanup temporary files to avoid https://github.com/helm/helm/issues/2977
cleanup() {
  if [[ -d "${KMCP_TMP_ROOT:-}" ]]; then
    rm -rf "$KMCP_TMP_ROOT"
  fi
}

# Execution

#Stop execution on any error
trap "fail_trap" EXIT
set -e

# Set debug if desired
if [ "${DEBUG}" == "true" ]; then
  set -x
fi

# Parsing input arguments (if any)
export INPUT_ARGUMENTS="${@}"
set -u
while [[ $# -gt 0 ]]; do
  case $1 in
    '--version'|-v)
       shift
       if [[ $# -ne 0 ]]; then
           export DESIRED_VERSION="${1}"
           if [[ "$1" != "v"* ]]; then
               echo "Expected version arg ('${DESIRED_VERSION}') to begin with 'v', fixing..."
               export DESIRED_VERSION="v${1}"
           fi
       else
           echo -e "Please provide the desired version. e.g. --version v0.1.0 or -v canary"
           exit 0
       fi
       ;;
    '--no-sudo')
       USE_SUDO="false"
       ;;
    '--help'|-h)
       help
       exit 0
       ;;
    *) exit 1
       ;;
  esac
  shift
done
set +u

initArch
initOS
verifySupported
checkDesiredVersion
if ! checkKmcpInstalledVersion; then
  downloadFile
  verifyFile
  installFile
fi
testVersion
addToPath
cleanup

echo ""
echo "🎉 KMCP installation completed successfully!"
echo ""
echo "Installation location: $KMCP_INSTALL_DIR/$BINARY_NAME"
echo ""
echo "To verify the installation, please run:"
echo "  kmcp --version"
echo ""
echo "KMCP has been added to your PATH, you may need to:"
echo "  - Restart your terminal, or"
echo "  - Run 'source ~/.zshrc' (for zsh) or 'source ~/.bashrc' (for bash)"
echo ""
echo "For more information, run:"
echo "  kmcp --help" 