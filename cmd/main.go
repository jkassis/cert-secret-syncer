package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	rt "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type SecretSyncer struct {
	client.Client
}

func (r *SecretSyncer) Reconcile(ctx context.Context, req rt.Request) (rt.Result, error) {
	log := log.FromContext(ctx)

	// Get the secret
	secret := &corev1.Secret{}
	if err := r.Get(ctx, req.NamespacedName, secret); err != nil {
		return rt.Result{}, client.IgnoreNotFound(err)
	}

	// Check annotations
	backend, ok := secret.Annotations["cert-secret-syncer/backend"]
	if !ok {
		return rt.Result{}, nil
	}

	switch backend {
	case "aws":
		// Create ACM service client
		awsSession := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
		awsAcmSvc := acm.New(awsSession)

		// Read certificate and key
		var cert, key []byte
		{
			cert, ok = secret.Data["cert"]
			if !ok {
				log.Error(fmt.Errorf("cert not found"), "Failed to sync cert")
				return rt.Result{}, fmt.Errorf("cert not found")
			}

			key, ok = secret.Data["key"]
			if !ok {
				log.Error(fmt.Errorf("key not found"), "Failed to sync cert")
				return rt.Result{}, fmt.Errorf("key not found")
			}
		}

		// Get certificate arn to update
		certificatArn := secret.Annotations["alb.ingress.kubernetes.io/certificate-arn"]

		// Import certificate to ACM
		{
			importCertInput := &acm.ImportCertificateInput{
				Certificate:    cert,
				CertificateArn: &certificatArn,
				PrivateKey:     key,
			}

			output, err := awsAcmSvc.ImportCertificate(importCertInput)
			if err != nil {
				log.Error(err, "Failed to import certificate")
				return rt.Result{}, err
			}

			// is this is the first import?
			if certificatArn == "" {
				// yes... save the ARN on the secret
				secret.Annotations["alb.ingress.kubernetes.io/certificate-arn"] = *output.CertificateArn
				err = r.Update(ctx, secret)
				if err != nil {
					log.Error(err, "Failed to update secret")
					return rt.Result{}, err
				}
			}

			certificatArn = *output.CertificateArn
		}

		// update the ingress with the certificate-arn
		var ingress *networkingv1.Ingress
		{
			ingressLabelsAsString, ok := secret.Annotations["cert-secret-syncer/ingress-labels"]
			if !ok {
				return rt.Result{}, nil
			}

			ingressLabels, err := r.parseLabels(ingressLabelsAsString)
			if err != nil {
				log.Error(err, "Failed to parse ingress labels")
				return rt.Result{}, err
			}

			ingress, err = r.getIngressByLabels(ctx, ingressLabels)
			if err != nil {
				log.Error(err, "ingress not found by labels")
				return rt.Result{}, err
			}

			ingress.Annotations["alb.ingress.kubernetes.io/certificate-arn"] = certificatArn
			err = r.Update(ctx, ingress)
			if err != nil {
				log.Error(err, "Failed to import certificate")
				return rt.Result{}, err
			}
		}

		// Handle other backends
	}

	return rt.Result{}, nil
}

func (r *SecretSyncer) getIngressByLabels(ctx context.Context, labelSet map[string]string) (*networkingv1.Ingress, error) {
	listOpts := &client.ListOptions{
		LabelSelector: labels.Set(labelSet).AsSelector(),
	}

	ingresses := &networkingv1.IngressList{}

	err := r.List(ctx, ingresses, listOpts)
	if err != nil {
		return nil, err
	}

	if len(ingresses.Items) == 0 {
		return nil, nil
	}

	return &ingresses.Items[0], nil

}

func (r *SecretSyncer) parseLabels(labelString string) (map[string]string, error) {
	labels := map[string]string{}

	pairs := strings.Split(labelString, ",")
	for _, pair := range pairs {
		kv := strings.Split(strings.TrimSpace(pair), "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid label spec: %q", pair)
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		if key == "" {
			return nil, fmt.Errorf("invalid label spec: %q", pair)
		}

		labels[key] = value
	}

	return labels, nil

}

func main() {
	check := func(err error) {
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	// get config for the controller-runtime
	config := rt.GetConfigOrDie()
	scheme := runtime.NewScheme()

	// Create cache
	cache, err := cache.New(config, cache.Options{})
	check(err)

	// Set up the controller manager
	mgr, err := rt.NewManager(config, rt.Options{
		Scheme: scheme,
	})
	check(err)

	c, err := controller.New("secret-syncer", mgr, controller.Options{
		Reconciler: &SecretSyncer{mgr.GetClient()},
	})
	check(err)

	err = c.Watch(source.Kind(cache, &corev1.Secret{}),
		&handler.EnqueueRequestForObject{})
	check(err)
}
