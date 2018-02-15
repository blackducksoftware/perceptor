#!/bin/bash
set -x

command=kubectl
project=perceptor-test-`date "+%Y-%m-%d-%H-%M-%s"`
#project=perceptor-test

executeTestFiles () {
  #oc project $project
  exitCode=100
  for filename in $(find . -type f -follow -print | grep -v perceptor-e2e.sh | xargs ls);
  do
    extension="${filename##*.}"
    echo "$filename : processing..."
    case "$extension" in
      sh)
          sh $filename $command $project
          exitCode=$?
          ;;
      yaml) echo "$filename : processing"
          $command create -f $filename -n $project
          exitCode=$?
          ;;
      *)  echo "invalid file [ $filename ] found in the test templates directory!"
          exit 1
          ;;
    esac
    if [[ $exitCode -gt 0 ]]; then
       echo "Test failed! $exitCode, exiting now."
       exit $exitCode
    fi
  done
}

if [[ $command -eq 'oc' ]]; then
  $command adm new-project $project
else
  $command create ns $project
fi

executeTestFiles

if [[ $command -eq 'oc' ]]; then
  $command delete project $project
else
  $command delete ns $project
fi
