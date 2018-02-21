#!/bin/bash

pushd ../../contrib/install/openshift
	oc delete project bds-perceptor
	while oc get ns | grep -q bds-perceptor ; do
		echo "waiting for delete"
		sleep 2
        done

	./deploy-openshift.sh
	if [[ $? -gt 0 ]] ; then
		echo "tests failed. bye"
		exit $?
	fi
popd

echo "passed !"
exit 0
