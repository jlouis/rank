#!/bin/sh

MAPS='aerowalk battleforged bloodrun campgrounds cure furiousheights hektik houseofdecay lostworld sinister toxicity verticalvengeance'
TOURNEYS=149
PASSWD="$1"

for i in $MAPS; do
	./rank -outfile="results/$i" -map=$i -passwd="$PASSWD" -tourneys=$TOURNEYS;
done

./rank -outfile="results/all" -passwd="$PASSWD" -tourneys=$TOURNEYS;
