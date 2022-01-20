#!/bin/sh



command_exists() {
	command -v "$@" > /dev/null 2>&1
}

is_linux() {
    case "$(uname -s)" in
    *linux* ) true ;;
    *Linux* ) true ;;
    * ) false;;
    esac
}

is_darwin() {
	case "$(uname -s)" in
	*darwin* ) true ;;
	*Darwin* ) true ;;
	* ) false;;
	esac
}

is_64arch() {
	case "$(uname -m)" in
	*64* ) true ;;
	* ) false;;
	esac
}

can_install() {
    if ! is_linux && ! is_darwin; then
        echo '# This script only supports Linux and MacOS'
        exit 1
    fi
    if ! is_64arch; then
        echo "# nhctl only supports 64-bit OS Systems"
        exit 1
    fi
}

do_download() {
    echo "# Executing nhctl download script"

    can_install

    DOWNLOAD_URL=$(curl -s https://github.com/nocalhost/nocalhost/releases/latest | cut  -d'"' -f2 | sed 's/tag/download/')
    if is_darwin; then
        DOWNLOAD_URL=${DOWNLOAD_URL}/nhctl-darwin-amd64
    fi
    if is_linux; then
        DOWNLOAD_URL=${DOWNLOAD_URL}/nhctl-linux-amd64
    fi

    curl -L $DOWNLOAD_URL -o nhctl
}

do_download