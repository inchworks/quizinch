# Quiz setup
The system holds settings for a single quiz. Specify:
- A title for the quiz.
- The name of the organiser.
- The number of rounds at the end that are tie-break rounds. For normal rounds, the scorer must enter a score for every team; for tie-breaks, the scorer enters scores just for the teams participating in the tie-break.
- The number of teams to be shown on the final leaderboard. (With a quiz for school children you might prefer to avoid highlighting the teams that came last.)
- The number of rounds for which answers and scoring are deferred. For example, if deferred by 1, the display will show round 1 questions, immediately followed by round 2 questions, then the answers and scores for round 1. This can speed up the quiz by allowing a round to be scored while teams are working on the questions for the next round.
- The frequency of updates to puppet displays, following changes by the controller. A sliding scale is used: 0 = 1/2s, 1 = 1s, 2 = 2s, 3 = 4s, 4 = 8s, etc. (Ideally puppet displays should update almost immediately, but with a slow network it may be better for each display device to poll for updates less often.)

## Teams
All that is needed for each team is a name. There is no fixed limit on the number of teams, but the scores slides may be less readable with more than 16 teams.

## Rounds
Each round has an order number, a title and optional format. Change the order numbers to re-order rounds after they have been set up.

Round formats modify the appearance of a round. Separate multiple formats by “\|”. E.g. “Q3\|A3\|I”.

| Code | Format |
| ---- | --- |
| Qn   | Limit the number of (Q)uestions per slide. E.g. “Q3” for 3 questions per slide, for long questions. |
| An   | Limit the number of (A)nswers per slide. E.g. “A3” for 3 answers per slide, when questions have long answers. |
| Cn   | (C)ombined questions and answers on a single page. Typically used for “sudden death” rounds. E.g. “C3”. |
| E    | Show (E)nd of quiz slide after round scores. |
| I    | Show (I)nterval slide after round scores. |

Notes
- An interval slide is shown after the scores. So for a 10 round quiz with an interval in the middle and deferred scoring, set “I” on round 4, because there will be questions for round 5, then answers and scores for round 4, and then the interval.
- Specify an end-of-quiz slide after the last normal round, and after each tie-break round.


## Questions
A round has a number of questions, each with an order number, question text, answer text and an optional media file.

Change the order numbers to re-order questions after they have been set up.

A question may include a picture, a sound clip, or a video clip for the media. Usually questions of one type would be grouped together into a picture round, a music round, etc. However this is not required.
- Audio files can be uploaded as mp3, aac, flac, or m4a.
- Video files can be uploaded as mp4 or mov.
- All other file types are assumed to be pictures. jpg, jpeg, and png are recommended.

## Changes during the quiz
You should not make changes during the quiz. However these changes can be made without the quiz restarting:
- You can delete a team, if they have to leave the quiz.
- (Is it possible to change a round's name?).
- (Is it possible to change a question or answer?).
