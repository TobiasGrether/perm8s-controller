#!/bin/bash
controller-gen crd paths="./pkg/apis/perm8s/v1alpha1" output:crd:artifacts:config=config/crd/bases
