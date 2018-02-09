.DEFAULT_GOAL := compile

.DEFAULT:
	for p in kube-perceiver perceptor perceptor-scanner; do \
		(echo $$(pwd)/cmd/$$p; cd $$(pwd)/cmd/$$p; make $@) ; \
	done;

test:
	go test ./pkg/...
