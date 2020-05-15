#!/bin/sh

if [ "$#" -lt 6 ]; then
  echo "Usage: ./client.sh -s <server> -f <filter file name> -m <message file name> [-p <patterns file name> -d <patterns directories>] [-c <input codec>]" >&2
  exit 1
fi

while getopts ":f:m:s:p:d:c:" opt; do
  case $opt in
    s) server="$OPTARG"
    ;;
    f) filter_file="$OPTARG"
    ;;
    m) message_file="$OPTARG"
    ;;
    p) patterns_file="$OPTARG"
    ;;
    d) patterns_dir="$OPTARG"
    ;;
    c) codec="$OPTARG"
    ;;
    \?) echo "Invalid option: -$OPTARG, call without args." >&2 && exit 1
    ;;
  esac
done

if [ ! -f "$filter_file" ]; then
    echo "$filter_file not exist"
    exit 1
fi

if [ ! -f "$message_file" ]; then
    echo "$message_file not exist"
    exit 1
fi

if ! [ -x "$(command -v curl)" ]; then
    echo "Curl not installed"
    exit 1
fi

ping_res=$(curl -s "$server"/ping)

if [ "$ping_res" != "pong" ]; then
	echo "Server $server is unavailable"
	exit 1
fi

if [ ! -f "$patterns_file" ] && [ -z "$codec" ]; then
  curl --request POST -F "filter=@$filter_file" -F "message=@$message_file" "$server"/upload && echo
fi

if [ ! -f "$patterns_file" ] && [ ! -z "$codec" ]; then
  curl --request POST -F "filter=@$filter_file" -F "message=@$message_file" -F "codec=$codec" "$server"/upload && echo
fi

if [ -f "$patterns_file" ] && [ -z "$codec" ]; then
  curl --request POST -F "filter=@$filter_file" -F "message=@$message_file" -F "patterns=@$patterns_file" -F "patterns_dir=$patterns_dir" "$server"/upload && echo
fi

if [ -f "$patterns_file" ] && [ ! -z "$codec" ]; then
  curl --request POST -F "filter=@$filter_file" -F "message=@$message_file" -F "codec=$codec" -F "patterns=@$patterns_file" -F "patterns_dir=$patterns_dir" "$server"/upload && echo
fi
