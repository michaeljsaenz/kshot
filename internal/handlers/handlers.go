package handlers

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/michaeljsaenz/probe/internal/k8s"
	"github.com/michaeljsaenz/probe/internal/network"
	"github.com/michaeljsaenz/probe/internal/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func StaticFiles(httpFS embed.FS) {
	types.UpdateSharedContextFS(httpFS)

	http.Handle("/static/", http.FileServer(http.FS(httpFS)))
}

func RootTemplate(w http.ResponseWriter, r *http.Request) {
	var fs embed.FS

	// retrieve embed.FS from shared context
	customValues, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValues.HttpFS
	}

	// use embed.FS to read/parse the template file
	tmpl := template.Must(template.ParseFS(fs, "templates/index.gohtml"))

	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func NetworkTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	var fs embed.FS

	// retrieve embed.FS from shared context
	customValues, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValues.HttpFS
	}

	// use embed.FS to read/parse the template file
	tmpl := template.Must(template.ParseFS(fs, "templates/network.gohtml"))

	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func IstioTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	var fs embed.FS
	var err error

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	tmpl := template.Must(template.ParseFS(fs, "templates/istio.gohtml"))

	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func KubernetesTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	var fs embed.FS
	var err error

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	application := types.NewApplication(types.Application{Error: err})

	tmpl := template.Must(template.ParseFS(fs, "templates/kubernetes.gohtml", "templates/main-*.gohtml", "templates/k8s/*.gohtml"))

	err = tmpl.Execute(w, application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ButtonSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	var err error
	var fs embed.FS
	var serverHeader, requestedUrlHeader, requestHostnameHeader, requestResponseStatus string

	url := r.PostFormValue("url")

	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")

	if url != "" {
		serverHeader, requestedUrlHeader, requestHostnameHeader, requestResponseStatus, err = network.GetUrl(url)
		if err != nil {
			log.Printf("error: %v", err)
		}
	}

	// update shared context with the requestedUrlHeader
	types.UpdateSharedContext(requestedUrlHeader, requestHostnameHeader)

	application := types.NewApplication(types.Application{HttpServerHeader: serverHeader,
		HttpRequestedUrl: requestHostnameHeader, HttpResponseStatus: requestResponseStatus, Error: err})

	if err == nil && url != "" {
		application.HttpRequestedUrl = "HTTP Requested URL: " + requestedUrlHeader
		application.HttpResponseStatus = "HTTP Response status: " + requestResponseStatus
		application.HttpServerHeader = "HTTP Server header: " + serverHeader
	}

	// retrieve embed.FS from shared context
	customValues, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValues.HttpFS
	}

	tmpl := template.Must(template.ParseFS(fs, "templates/network.gohtml"))

	err = tmpl.ExecuteTemplate(w, "submit", application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func ButtonCertificates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	types.ContextLock.Lock()
	defer types.ContextLock.Unlock()

	var err error
	var certificates string

	// retrieve value from shared context
	customValues, ok := types.SharedContext.Value(types.ContextKey).(types.CustomContextValues)
	if ok && customValues.URL != "" {
		certificates, err = network.GetCertificates(customValues.URL)
		if err != nil {
			log.Printf("error: %v", err)
		}
	}

	application := types.NewApplication(types.Application{HttpRequestedUrl: customValues.URL,
		Certificates: certificates, Error: err})

	var fs embed.FS

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	tmpl := template.Must(template.ParseFS(fs, "templates/network.gohtml"))

	err = tmpl.ExecuteTemplate(w, "certificates", application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ButtonDNS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	types.ContextLock.Lock()
	defer types.ContextLock.Unlock()

	var err error
	var ips []string

	// retrieve value from shared context
	customValues, ok := types.SharedContext.Value(types.ContextKey).(types.CustomContextValues)
	if ok && customValues.Hostname != "" {
		ips, err = network.GetDNS(customValues.Hostname)
		if err != nil {
			log.Printf("error: %v", err)
		}
	}

	application := types.NewApplication(types.Application{DNS: ips, Error: err})

	var fs embed.FS

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	tmpl := template.Must(template.ParseFS(fs, "templates/network.gohtml"))

	err = tmpl.ExecuteTemplate(w, "dns-lookup", application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func ButtonPing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	types.ContextLock.Lock()
	defer types.ContextLock.Unlock()

	var err error
	var pingResponse string
	var pingResponses []string

	// retrieve value from shared context
	customValues, ok := types.SharedContext.Value(types.ContextKey).(types.CustomContextValues)
	if ok && customValues.Hostname != "" {
		for i := 0; i < 4; i++ {
			pingResponse, err = network.Ping(customValues.Hostname)
			if err != nil {
				log.Printf("error: %v", err)
			}
			pingResponses = append(pingResponses, pingResponse)
		}

	}

	application := types.NewApplication(types.Application{PingResponses: pingResponses, Error: err})

	var fs embed.FS

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	tmpl := template.Must(template.ParseFS(fs, "templates/network.gohtml"))

	err = tmpl.ExecuteTemplate(w, "ping", application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ButtonTraceroute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	types.ContextLock.Lock()
	defer types.ContextLock.Unlock()

	var err error
	var tracerouteResult string

	// retrieve value from shared context
	customValues, ok := types.SharedContext.Value(types.ContextKey).(types.CustomContextValues)
	if ok && customValues.Hostname != "" {
		tracerouteResult, err = network.Traceroute(customValues.Hostname)
		if err != nil {
			log.Printf("%v", err)
		}
	}

	application := types.NewApplication(types.Application{TracerouteResult: tracerouteResult, Error: err})

	var fs embed.FS

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	tmpl := template.Must(template.ParseFS(fs, "templates/network.gohtml"))

	err = tmpl.ExecuteTemplate(w, "traceroute", application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func DropdownNamespaceSelection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	var clientset *kubernetes.Clientset
	var config *rest.Config

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusInternalServerError)
		return
	}

	namespace := r.FormValue("namespace")

	//retrieve k8s clientset from shared context
	customValues, ok := types.SharedContextK8s.Value(types.ContextKey).(types.CustomContextValuesK8s)
	if ok {
		clientset = customValues.Clientset
		config = customValues.Config
	}

	types.UpdateSharedContextK8s(clientset, config, namespace, "")

	application := types.NewApplication(types.Application{K8sSelectedNamespace: namespace, Error: err})

	fmt.Fprintf(w, "%s", application.K8sSelectedNamespace)

}

func ButtonGetPods(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	var clientset *kubernetes.Clientset
	var namespace string
	var fs embed.FS

	//retrieve k8s clientset from shared context
	customValues, ok := types.SharedContextK8s.Value(types.ContextKey).(types.CustomContextValuesK8s)
	if ok {
		clientset = customValues.Clientset
		namespace = customValues.Namespace
	}

	pods, err := k8s.GetPodsInNamespace(clientset, namespace)
	if err != nil {
		log.Printf("error: %v", err)
	}

	application := types.NewApplication(types.Application{K8sPods: pods, Error: err})

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	tmpl := template.Must(template.ParseFS(fs, "templates/kubernetes.gohtml", "templates/k8s/*.gohtml"))

	err = tmpl.ExecuteTemplate(w, "get-pods", application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ButtonGetNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	var clientset *kubernetes.Clientset
	var fs embed.FS

	//retrieve k8s clientset from shared context
	customValues, ok := types.SharedContextK8s.Value(types.ContextKey).(types.CustomContextValuesK8s)
	if ok {
		clientset = customValues.Clientset
	}

	nodes, nodesDetail, err := k8s.GetNodes(clientset)
	if err != nil {
		log.Printf("error: %v", err)
	}

	application := types.NewApplication(types.Application{K8sNodes: nodes, K8sNodesDetail: nodesDetail, Error: err})

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	tmpl := template.Must(template.ParseFS(fs, "templates/kubernetes.gohtml", "templates/k8s/*.gohtml"))

	err = tmpl.ExecuteTemplate(w, "get-nodes", application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func ButtonGetNodeConditions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	var clientset *kubernetes.Clientset
	var fs embed.FS
	var k8sNode types.K8sNode
	var err error

	//retrieve k8s clientset from shared context
	customValues, ok := types.SharedContextK8s.Value(types.ContextKey).(types.CustomContextValuesK8s)
	if ok {
		clientset = customValues.Clientset
	}

	// directly from form
	node := strings.TrimSpace(r.PostFormValue("node"))
	if node == "" {
		k8sNode.Name = node
	} else {
		k8sNode, err = k8s.GetNodeConditions(clientset, node)
		if err != nil {
			log.Printf("error: %v", err)
		}
	}

	application := types.NewApplication(types.Application{K8sNode: k8sNode, Error: err})

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	tmpl := template.Must(template.ParseFS(fs, "templates/kubernetes.gohtml", "templates/k8s/*.gohtml"))

	err = tmpl.ExecuteTemplate(w, "get-node-conditions", application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func ButtonGetNamespaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	var fs embed.FS
	var err error
	var clientset *kubernetes.Clientset

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	// retrieve k8s clientset from shared context
	customValues, ok := types.SharedContextK8s.Value(types.ContextKey).(types.CustomContextValuesK8s)
	if ok {
		clientset = customValues.Clientset

	}

	// retrieve namespaces
	namespaces, err := k8s.GetNamespaces(clientset)
	if err != nil {
		log.Printf("error: %v", err)
	}
	allNamespaces := "All Namespaces"
	namespaces = append(namespaces, allNamespaces)

	application := types.NewApplication(types.Application{K8sNamespaces: namespaces, Error: err})

	tmpl := template.Must(template.ParseFS(fs, "templates/kubernetes.gohtml", "templates/k8s/*.gohtml"))

	err = tmpl.ExecuteTemplate(w, "get-namespaces", application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ButtonPodDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	var err error
	var podDetail types.PodDetail
	var fs embed.FS
	var clientset *kubernetes.Clientset
	var config *rest.Config

	// directly from form
	pod := strings.TrimSpace(r.PostFormValue("pod"))
	namespace := strings.TrimSpace(r.PostFormValue("namespace"))

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	//retrieve k8s clientset from shared context
	customValues, ok := types.SharedContextK8s.Value(types.ContextKey).(types.CustomContextValuesK8s)
	if ok {
		clientset = customValues.Clientset
		config = customValues.Config
		if namespace == "" {
			namespace = customValues.Namespace
		}
		if pod == "" {
			pod = customValues.Pod
		}
	}

	if pod != "" {
		types.UpdateSharedContextK8s(clientset, config, namespace, pod)
	}

	if pod != "" {
		podDetail, err = k8s.GetPodDetail(clientset, namespace, pod)
		if err != nil {
			log.Printf("error: %v", err)
		}
	}

	application := types.NewApplication(types.Application{K8sPodDetail: podDetail, Error: err})

	tmpl := template.Must(template.ParseFS(fs, "templates/kubernetes.gohtml", "templates/k8s/*.gohtml"))

	err = tmpl.ExecuteTemplate(w, "get-pod-details", application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func ClickPodYaml(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	var err error
	var namespace string
	var podYaml string
	var fs embed.FS
	var clientset *kubernetes.Clientset

	pod := strings.TrimSpace(r.PostFormValue("pod"))

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	//retrieve k8s clientset from shared context
	customValues, ok := types.SharedContextK8s.Value(types.ContextKey).(types.CustomContextValuesK8s)
	if ok {
		clientset = customValues.Clientset
		namespace = customValues.Namespace

	}

	if pod != "" {
		podYaml, err = k8s.GetPodYaml(clientset, namespace, pod)
		if err != nil {
			log.Printf("error: %v", err)
		}
	}

	application := types.NewApplication(types.Application{K8sPodYaml: podYaml, Error: err})

	tmpl := template.Must(template.ParseFS(fs, "templates/kubernetes.gohtml", "templates/k8s/*.gohtml"))

	err = tmpl.ExecuteTemplate(w, "get-pod-yaml", application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func ClearContextK8sNamespace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	var clientset *kubernetes.Clientset
	var config *rest.Config

	// retrieve k8s clientset from shared context
	customValues, ok := types.SharedContextK8s.Value(types.ContextKey).(types.CustomContextValuesK8s)
	if ok {
		clientset = customValues.Clientset
		config = customValues.Config
	}

	types.UpdateSharedContextK8s(clientset, config, "", "")

}

func ClickContainerLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var fs embed.FS
	var clientset *kubernetes.Clientset

	pod := strings.TrimSpace(r.PostFormValue("pod"))
	container := strings.TrimSpace(r.PostFormValue("container"))
	namespace := strings.TrimSpace(r.PostFormValue("namespace"))

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	//retrieve k8s clientset from shared context
	customValues, ok := types.SharedContextK8s.Value(types.ContextKey).(types.CustomContextValuesK8s)
	if ok {
		clientset = customValues.Clientset
		if namespace == "" {
			namespace = customValues.Namespace
		}
	}

	podLog, err := k8s.GetContainerLog(clientset, pod, container, namespace)

	application := types.NewApplication(types.Application{K8sPodLog: podLog, Error: err})

	tmpl := template.Must(template.ParseFS(fs, "templates/kubernetes.gohtml", "templates/main-display.gohtml", "templates/k8s/*.gohtml"))

	err = tmpl.ExecuteTemplate(w, "get-container-log", application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func ClickContainerPort(w http.ResponseWriter, r *http.Request) {
	var clientset *kubernetes.Clientset
	var config *rest.Config
	var fs embed.FS

	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	pod := strings.TrimSpace(r.PostFormValue("pod"))
	// container := strings.TrimSpace(r.PostFormValue("container"))
	containerPort := strings.TrimSpace(r.PostFormValue("port"))
	namespace := strings.TrimSpace(r.PostFormValue("namespace"))

	//retrieve k8s clientset from shared context
	customValues, ok := types.SharedContextK8s.Value(types.ContextKey).(types.CustomContextValuesK8s)
	if ok {
		clientset = customValues.Clientset
		config = customValues.Config
		// namespace from sharedContext empty if not selected from dropdown
	}

	url, podPort, err := k8s.PortForward(clientset, config, namespace, pod, containerPort)
	if err != nil {
		log.Printf("error: %v", err)
	}

	pf := types.K8sPodPortForward{
		URL:     url,
		PodPort: podPort,
	}

	application := types.NewApplication(types.Application{K8sPodPortForward: pf, Error: err})

	tmpl := template.Must(template.ParseFS(fs, "templates/kubernetes.gohtml", "templates/k8s/*.gohtml"))

	err = tmpl.ExecuteTemplate(w, "click-container-port", application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func SubmitContainerExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var fs embed.FS
	var clientset *kubernetes.Clientset
	var config *rest.Config
	var containerExecResponse string

	pod := strings.TrimSpace(r.PostFormValue("pod"))
	container := strings.TrimSpace(r.PostFormValue("container"))
	namespace := strings.TrimSpace(r.PostFormValue("namespace"))
	command := r.PostFormValue("command")

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	//retrieve k8s clientset from shared context
	customValues, ok := types.SharedContextK8s.Value(types.ContextKey).(types.CustomContextValuesK8s)
	if ok {
		clientset = customValues.Clientset
		config = customValues.Config
	}

	containerDetail := types.K8sContainerDetail{
		PodName:               pod,
		PodNamespace:          namespace,
		ContainerName:         container,
		ContainerExecResponse: containerExecResponse,
	}

	containerExecResponse, err := k8s.ContainerExec(clientset, config, pod, container, namespace, command)
	if err != nil {
		log.Printf("error: %v", err)
	}
	containerDetail.ContainerExecResponse = containerExecResponse

	application := types.NewApplication(types.Application{K8sContainerDetail: containerDetail, Error: err})

	tmpl := template.Must(template.ParseFS(fs, "templates/kubernetes.gohtml", "templates/main-display.gohtml", "templates/k8s/*.gohtml"))

	err = tmpl.ExecuteTemplate(w, "submit-container-exec", application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ClickContainerExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && !strings.Contains(r.Header.Get("HX-Request"), "true") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	var err error
	var fs embed.FS

	pod := strings.TrimSpace(r.PostFormValue("pod"))
	container := strings.TrimSpace(r.PostFormValue("container"))
	namespace := strings.TrimSpace(r.PostFormValue("namespace"))

	// retrieve embed.FS from shared context
	customValueFS, ok := types.SharedContextFS.Value(types.ContextKey).(types.CustomContextValuesFS)
	if ok {
		fs = customValueFS.HttpFS
	}

	containerDetail := types.K8sContainerDetail{
		PodName:       pod,
		PodNamespace:  namespace,
		ContainerName: container,
	}

	application := types.NewApplication(types.Application{K8sContainerDetail: containerDetail, Error: err})

	tmpl := template.Must(template.ParseFS(fs, "templates/kubernetes.gohtml", "templates/main-display.gohtml", "templates/k8s/*.gohtml"))

	err = tmpl.ExecuteTemplate(w, "click-container-exec", application)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
