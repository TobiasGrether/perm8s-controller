for filename in config/crd/bases; do
    kubectl apply -f $filename
done
