kind create cluster --config=config.yaml
kind load docker-image mongo go-mongo-client
kubectl apply -f completo.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait --namespace ingress-nginx ^
  --for=condition=ready pod ^
  --selector=app.kubernetes.io/component=controller ^
  --timeout=180s
kubectl apply -f ingress.yaml