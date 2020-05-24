#!/bin/bash

SCRIPT_PATH=${BASH_SOURCE[0]}
if [ -L "${SCRIPT_PATH}" ]
then
    SCRIPT_PATH=$(readlink $SCRIPT_PATH)
fi
SCRIPT_DIR="$( cd "$( dirname "${SCRIPT_PATH}" )" && pwd )"

"$SCRIPT_DIR/Deepl-translator-linux"

xdotool search --desktop 0 Deepl-translator-linux windowactivate
