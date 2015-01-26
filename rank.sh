#!/bin/sh

MAPS='aerowalk battleforged bloodrun campgrounds cure furiousheights hektik houseofdecay lostworld sinister silence toxicity verticalvengeance'
TOURNEYS=153
PASSWD="$1"

for i in $MAPS; do
	env GOMAXPROCS=8 ./rank -outfile="results/$i" -map=$i -passwd="$PASSWD" -tourneys=$TOURNEYS;
done

./rank -outfile="results/all" -passwd="$PASSWD" -tourneys=$TOURNEYS;
