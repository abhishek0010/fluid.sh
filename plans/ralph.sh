set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <iterations>"
    exit 1
fi

#     5. Make a git commit of that feature. \

for((i=1; i<=$1; i++)); do
    echo "----------------------------"
    result = $(claude --permission-mode acceptEdits -p "@plans/prd.json @plans/progress.txt \
    1. Find the highest-priority feature to work on and work only on that feature. \
    This should be the one YOU decide has the highest priority - not necessarily the first in the list. \
    2. Check that it builds via make build and that the tests pass via make test\
    3. Update the PRD with the work that was done.\
    4. Append your progress to the progress.txt file.\
    ONLY WORK ON A SINGLE FEATURE. \
    If , while implimenting the feature, you notice the PRD is complete, ouput <promise>COMPLETE</promise> \
    ")

    echo "result"

    if [[ "result" == *"<promise>COMPLETE</promise>" ]]; then
        echo "PRD complete, exiting"
        exit 0
    fi
done
