set -e

#     5. Make a git commit of that feature. \

claude --permission-mode acceptEdits -p "@plans/prd.json @plans/progress.txt \
1. Find the highest-priority feature to work on and work only on that feature. \
This should be the one YOU decide has the highest priority - not necessarily the first in the list. \
2. Check that it runs via uv run and that the tests pass via uv test\
3. Update the PRD with the work that was done.\
4. Append your progress to the @plans/progress.txt file.\
ONLY WORK ON A SINGLE FEATURE. \
If, while implimenting the feature, you notice the PRD is complete, ouput <promise>COMPLETE</promise> \
"
