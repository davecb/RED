# RED -- requests, errors and durations

An example of sharing memory by communicating, for a
common use case, reporting metrics.

Several larger-scale go metrics packages communicate by sharing memory,
and suffer from painful lock contention problems.

For the back-story, have a look at "Dave's Red Rant", in Red.md
