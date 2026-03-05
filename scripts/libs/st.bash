#!/bin/bash

BOLD=
OFFBOLD=
RESET_COLOR=
RED=
GREEN=
YELLOW=
BLUE=
BLUE_CYAN=
GRAY_LIGHT=

if [ -t 1 ]; then
    BOLD=$(tput bold)
    OFFBOLD=$(tput sgr0)
    RESET_COLOR="$(tput sgr0)"
    RED="$(
        tput bold
        tput setaf 1
    )"
    GREEN="$(
        tput bold
        tput setaf 2
    )"
    YELLOW="$(
        tput bold
        tput setaf 3
    )"
    BLUE="$(
        tput bold
        tput setaf 4
    )"
    BLUE_CYAN="$(
        tput bold
        tput setaf 6
    )"
    GRAY_LIGHT="$(tput setaf 250)"
fi

DOING_MSG=
ST_QUIET="${ST_QUIET:-false}"

function st.quiet() {
    ST_QUIET=true
}

function st.unquiet() {
    ST_QUIET=false
}

function st.cmd.exists() {
    command -v "$1" >/dev/null 2>&1
}

## Usage: st.var.exists A_VAR && echo PASS
function st.var.exists() {
    [ -n "${!1:-}" ]
}

function st.h1() {
    local prefix=
    [ "$ST_QUIET" != "true" ] && prefix='st.h1> '
    echo -e "${prefix}${BOLD}$1${OFFBOLD}"
}

function st.h2() {
    local prefix=
    [ "$ST_QUIET" != "true" ] && prefix='st.h2> '
    echo -e "${prefix}${BOLD}$1${OFFBOLD}"
}

function st.h3() {
    local prefix=
    [ "$ST_QUIET" != "true" ] && prefix='st.h3> '
    echo -e "${prefix}${BOLD}$1${OFFBOLD}"
}

function st.doing() {
    DOING_MSG=$1

    local prefix=
    [ "$ST_QUIET" != "true" ] && prefix='st.doing> '
    echo "${prefix}${BLUE}${DOING_MSG:-…}$RESET_COLOR"
}

function st.done() {
    local DONE="${1:-[DONE]}"

    local prefix=
    [ "$ST_QUIET" != "true" ] && prefix='st.done> '
    echo "${prefix}${DOING_MSG:-} : ${GREEN}$DONE${RESET_COLOR}"
}

function st.success() {
    local MSG="${1:-[SUCCESS]}"

    local prefix=
    [ "$ST_QUIET" != "true" ] && prefix='st.success> '
    echo "${prefix}${BOLD}${GREEN}${MSG}${RESET_COLOR}${OFFBOLD}"
}

function st.nothing() {
    local MSG="${1:-[NOTHING TO DO]}"
    local prefix=
    [ "$ST_QUIET" != "true" ] && prefix='st.nothingtd> '
    echo "${prefix}${DOING_MSG:-} : ${GREEN}${MSG}${RESET_COLOR}"
}

function st.skipped() {
    local MSG="${1:-[SKIPPED]}"
    local prefix=
    [ "$ST_QUIET" != "true" ] && prefix='st.skipped> '
    echo "${prefix}${DOING_MSG:-} : ${BLUE_CYAN}${MSG}${RESET_COLOR}"
}

function st.warn() {
    local prefix=
    [ "$ST_QUIET" != "true" ] && prefix='st.warn> '
    echo "${prefix}${BOLD}${YELLOW}$1${RESET_COLOR}${OFFBOLD}"
}

function st.fail() {
    local MSG="${1:-[FAILED]}"
    local prefix=
    [ "$ST_QUIET" != "true" ] && prefix='st.fail '
    echo -e "${prefix}${DOING_MSG:-} : ${RED}$MSG${RESET_COLOR}"
    false
}

function st.abort() {
    local MSG="${1:-[ABORTED]}"
    local prefix=
    [ "$ST_QUIET" != "true" ] && prefix='st.abort> '
    echo -e "${prefix}${DOING_MSG:-} : ${BOLD}${RED}${MSG}${RESET_COLOR}${OFFBOLD}\n"
    false

    exit 1
}

function st.do() {
    local -a cmd=("$@")
    local prefix=
    [ "$ST_QUIET" != "true" ] && prefix='st.do> '
    echo "${prefix}${GRAY_LIGHT}${cmd[*]}${RESET_COLOR}"
    "${cmd[@]}"
}
