#!/bin/sh
set -e

mkdir -p /tmp/hls_booster /tmp/hls_bridge /tmp/hls_night

echo "Transcoding neon-booster..."
ffmpeg -y -i /tmp/neon-booster.wav -filter:a loudnorm -map 0:a -map 0:a -c:a aac -b:a:0 128k -b:a:1 256k -ar:0 44100 -ar:1 48000 -ac:0 2 -ac:1 2 -f hls -hls_time 6 -hls_playlist_type vod -hls_segment_filename /tmp/hls_booster/seg_%v_%03d.ts -master_pl_name master.m3u8 -var_stream_map "a:0 a:1" /tmp/hls_booster/stream_%v.m3u8

echo "Transcoding neon-bridge-run..."
ffmpeg -y -i /tmp/neon-bridge-run.wav -filter:a loudnorm -map 0:a -map 0:a -c:a aac -b:a:0 128k -b:a:1 256k -ar:0 44100 -ar:1 48000 -ac:0 2 -ac:1 2 -f hls -hls_time 6 -hls_playlist_type vod -hls_segment_filename /tmp/hls_bridge/seg_%v_%03d.ts -master_pl_name master.m3u8 -var_stream_map "a:0 a:1" /tmp/hls_bridge/stream_%v.m3u8

echo "Transcoding night-view..."
ffmpeg -y -i /tmp/night-view.wav -filter:a loudnorm -map 0:a -map 0:a -c:a aac -b:a:0 128k -b:a:1 256k -ar:0 44100 -ar:1 48000 -ac:0 2 -ac:1 2 -f hls -hls_time 6 -hls_playlist_type vod -hls_segment_filename /tmp/hls_night/seg_%v_%03d.ts -master_pl_name master.m3u8 -var_stream_map "a:0 a:1" /tmp/hls_night/stream_%v.m3u8

echo "All transcoded!"
