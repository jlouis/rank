#!/bin/sh

MAPS='aerowalk battleforged bloodrun campgrounds cure furiousheights hektik houseofdecay lostworld sinister toxicity verticalvengeance'
TOURNEYS=5

for i in $MAPS; do
	./rank -outfile="results/$i" -map=$i -passwd="$1" -tourneys=$TOURNEYS;
done

./rank -outfile="results/all" -passwd="$1" -tourneys=$TOURNEYS;
