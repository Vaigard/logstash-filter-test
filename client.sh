#!/bin/sh

if [ "$#" -ne 6 ]; then
  echo "Usage: ./client.sh -s <server> -f <filter file name> -m <message file name>" >&2
  exit 1
fi

while getopts ":f:m:s:" opt; do
  case $opt in
  	s) server="$OPTARG"
    ;;
    f) filter_file="$OPTARG"
    ;;
    m) message_file="$OPTARG"
    ;;
    \?) echo "Invalid option: -$OPTARG" >&2 && exit 1
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

echo "Start testing..."

curl -i --request POST -F "filter=@$filter_file" -F "message=@$message_file" "$server"/upload && echo
