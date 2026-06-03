package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/proxyops"
	"github.com/spf13/cobra"
)

func defaultAPIPaths() []string {
	return []string{
		"/api", "/sanctum", "/broadcasting", "/storage",
		"/redirect",     // Core/Routes/web.php GET /redirect/{profile?} → entrypoint SSO (alvo do 401 da SPA)
		"/authenticate", // Core/Routes/web.php GET /authenticate/{profile?} + api.php POST → callback SSO
		"/login", "/logout", "/up", // convenções Laravel/Breeze + healthcheck
	}
}

// buildAPIRoutes turns the opinionated CLI flags into config.Route entries.
// Exactly one of apiSite/apiPort may be set. When neither is set it returns
// (nil, nil) — a simple proxy. Paths default to defaultAPIPaths().
func buildAPIRoutes(apiSite string, apiPort int, apiPaths []string) ([]config.Route, error) {
	hasSite := apiSite != ""
	hasPort := apiPort != 0
	if !hasSite && !hasPort {
		if len(apiPaths) > 0 {
			return nil, fmt.Errorf("--api-path requer --api-site ou --api-port")
		}
		return nil, nil
	}
	if hasSite && hasPort {
		return nil, fmt.Errorf("use --api-site OU --api-port, não os dois")
	}
	paths := apiPaths
	if len(paths) == 0 {
		paths = defaultAPIPaths()
	}
	var routes []config.Route
	for _, p := range paths {
		r := config.Route{Path: p}
		if hasSite {
			r.Site = apiSite
		} else {
			r.UpstreamPort = apiPort
		}
		routes = append(routes, r)
	}
	return routes, nil
}

// fullstackHint returns an advisory message printed after creating/editing a
// fullstack proxy so the user knows the API site's .env was pointed at the
// unified domain.
func fullstackHint(domain string, secured bool) string {
	scheme := "http"
	if secured {
		scheme = "https"
	}
	return fmt.Sprintf("dica: o .env do site de API foi apontado para %s://%s "+
		"(APP_URL / SANCTUM_STATEFUL_DOMAINS, se presentes). Ajuste manualmente se necessário.", scheme, domain)
}

// NewProxyCmd is the root of `lerd proxy …`.
func NewProxyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Gerenciar proxies para projetos não-PHP (SPAs, dev servers)",
		Long:  "Registra um domínio local que faz reverse proxy para um dev server arbitrário no host (Vue/Quasar/Nuxt/Vite). HTTPS e WebSocket habilitados por padrão.",
	}
	cmd.AddCommand(newProxyAddCmd())
	cmd.AddCommand(newProxyEditCmd())
	cmd.AddCommand(newProxyLsCmd())
	cmd.AddCommand(newProxyRmCmd())
	cmd.AddCommand(newProxySecureCmd(true))
	cmd.AddCommand(newProxySecureCmd(false))
	cmd.AddCommand(newProxyPauseCmd(true))
	cmd.AddCommand(newProxyPauseCmd(false))
	cmd.AddCommand(newProxyStartCmd())
	cmd.AddCommand(newProxyStopCmd())
	cmd.AddCommand(newProxyLogsCmd())
	return cmd
}

func newProxyAddCmd() *cobra.Command {
	var port int
	var path string
	var noSecure bool
	var managed bool
	var cmdStr string
	var nodeVersion string
	var autostart bool
	var apiSite string
	var apiPort int
	var apiPaths []string

	c := &cobra.Command{
		Use:   "add <domínio>",
		Short: "Registrar um novo proxy",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			routes, rerr := buildAPIRoutes(apiSite, apiPort, apiPaths)
			if rerr != nil {
				return rerr
			}
			p, err := proxyops.Add(proxyops.AddOptions{
				Domain:      strings.ToLower(args[0]),
				Port:        port,
				Path:        path,
				NoSecure:    noSecure,
				Managed:     managed,
				Command:     cmdStr,
				NodeVersion: nodeVersion,
				AutoStart:   autostart,
				Routes:      routes,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Proxy criado: %s → %s:%d (secured=%v, managed=%v)\n",
				p.PrimaryDomain(), upstreamForDisplay(p), p.UpstreamPort, p.Secured, p.Managed)
			if len(routes) > 0 {
				fmt.Println(fullstackHint(p.PrimaryDomain(), p.Secured))
			}
			return nil
		},
	}
	c.Flags().IntVar(&port, "port", 0, "Porta do upstream (obrigatória)")
	c.Flags().StringVar(&path, "path", "", "Pasta do projeto (obrigatória se --managed)")
	c.Flags().BoolVar(&noSecure, "no-secure", false, "Cria como HTTP em vez de HTTPS")
	c.Flags().BoolVar(&managed, "managed", false, "lerd gerencia o dev server via quadlet")
	c.Flags().StringVar(&cmdStr, "cmd", "", "Comando para iniciar o dev server (ex: 'npm run dev')")
	c.Flags().StringVar(&nodeVersion, "node", "", "Major version do Node (ex: '20'); default: 20")
	c.Flags().BoolVar(&autostart, "autostart", false, "Iniciar com `lerd start`")
	c.Flags().StringVar(&apiSite, "api-site", "", "Site do lerd que serve a API (fullstack)")
	c.Flags().IntVar(&apiPort, "api-port", 0, "Porta da API em dev server externo (fullstack)")
	c.Flags().StringArrayVar(&apiPaths, "api-path", nil, "Path roteado para a API (repetível; default: /api /sanctum /broadcasting /storage)")
	_ = c.MarkFlagRequired("port")
	return c
}

func newProxyEditCmd() *cobra.Command {
	var port int
	var path string
	var cmdStr string
	var nodeVersion string
	var upstreamHost string
	var autostart bool
	var apiSite string
	var apiPort int
	var apiPaths []string

	c := &cobra.Command{
		Use:   "edit <nome-ou-domínio>",
		Short: "Editar campos de um proxy existente (sem trocar domínio/managed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := resolveProxyName(args[0])
			opts := proxyops.UpdateOptions{}
			if cmd.Flags().Changed("port") {
				opts.Port = &port
			}
			if cmd.Flags().Changed("path") {
				opts.Path = &path
			}
			if cmd.Flags().Changed("cmd") {
				opts.Command = &cmdStr
			}
			if cmd.Flags().Changed("node") {
				opts.NodeVersion = &nodeVersion
			}
			if cmd.Flags().Changed("upstream-host") {
				opts.UpstreamHost = &upstreamHost
			}
			if cmd.Flags().Changed("autostart") {
				opts.AutoStart = &autostart
			}
			if cmd.Flags().Changed("api-site") || cmd.Flags().Changed("api-port") || cmd.Flags().Changed("api-path") {
				routes, rerr := buildAPIRoutes(apiSite, apiPort, apiPaths)
				if rerr != nil {
					return rerr
				}
				opts.Routes = &routes
			}
			p, err := proxyops.Update(name, opts)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Proxy %s atualizado: %s → %s:%d\n",
				p.Name, p.PrimaryDomain(), upstreamForDisplay(*p), p.UpstreamPort)
			if opts.Routes != nil && len(*opts.Routes) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), fullstackHint(p.PrimaryDomain(), p.Secured))
			}
			return nil
		},
	}
	c.Flags().IntVar(&port, "port", 0, "Nova porta do upstream")
	c.Flags().StringVar(&path, "path", "", "Nova pasta do projeto (string vazia limpa)")
	c.Flags().StringVar(&cmdStr, "cmd", "", "Novo comando do dev server (managed)")
	c.Flags().StringVar(&nodeVersion, "node", "", "Novo major do Node (managed)")
	c.Flags().StringVar(&upstreamHost, "upstream-host", "", "Hostname alternativo do upstream (default: host.containers.internal)")
	c.Flags().BoolVar(&autostart, "autostart", false, "Iniciar com `lerd start` (managed)")
	c.Flags().StringVar(&apiSite, "api-site", "", "Site do lerd que serve a API (fullstack)")
	c.Flags().IntVar(&apiPort, "api-port", 0, "Porta da API em dev server externo (fullstack)")
	c.Flags().StringArrayVar(&apiPaths, "api-path", nil, "Path roteado para a API (repetível; default: /api /sanctum /broadcasting /storage)")
	return c
}

func newProxyLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "Listar proxies",
		RunE: func(cmd *cobra.Command, _ []string) error {
			reg, err := config.LoadProxies()
			if err != nil {
				return err
			}
			if len(reg.Proxies) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Nenhum proxy registrado. Use `lerd proxy add` para criar um.")
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 2, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tDOMAIN\tPORT\tTLS\tMANAGED\tPATH")
			for _, p := range reg.Proxies {
				tls := "no"
				if p.Secured {
					tls = "yes"
				}
				mgr := "no"
				if p.Managed {
					mgr = "yes"
				}
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
					p.Name, p.PrimaryDomain(), p.UpstreamPort, tls, mgr, p.Path)
			}
			return w.Flush()
		},
	}
}

func newProxyRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <nome-ou-domínio>",
		Short: "Remover um proxy",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := resolveProxyName(args[0])
			if err := proxyops.Remove(name); err != nil {
				return err
			}
			fmt.Printf("Proxy %s removido.\n", name)
			return nil
		},
	}
}

func newProxySecureCmd(secured bool) *cobra.Command {
	use := "secure <nome-ou-domínio>"
	short := "Habilitar HTTPS via mkcert"
	if !secured {
		use = "unsecure <nome-ou-domínio>"
		short = "Desabilitar HTTPS"
	}
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := config.FindProxy(resolveProxyName(args[0]))
			if err != nil {
				return err
			}
			if err := proxyops.SetSecured(p, secured); err != nil {
				return err
			}
			fmt.Printf("Proxy %s agora é %s.\n", p.Name, scheme(secured))
			return nil
		},
	}
}

func newProxyPauseCmd(pause bool) *cobra.Command {
	use := "pause <nome-ou-domínio>"
	short := "Pausar o proxy (remove vhost mas mantém entry)"
	if !pause {
		use = "resume <nome-ou-domínio>"
		short = "Reativar o proxy pausado"
	}
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := config.FindProxy(resolveProxyName(args[0]))
			if err != nil {
				return err
			}
			p.Paused = pause
			if err := proxyops.ApplyPause(p); err != nil {
				return err
			}
			verb := "pausado"
			if !pause {
				verb = "reativado"
			}
			fmt.Printf("Proxy %s %s.\n", p.Name, verb)
			return nil
		},
	}
}

func resolveProxyName(input string) string {
	if p, err := config.FindProxyByDomain(input); err == nil && p != nil {
		return p.Name
	}
	return input
}

func upstreamForDisplay(p config.Proxy) string {
	if p.UpstreamHost != "" {
		return p.UpstreamHost
	}
	return "host.containers.internal"
}

func scheme(secured bool) string {
	if secured {
		return "HTTPS"
	}
	return "HTTP"
}

func newProxyStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <nome-ou-domínio>",
		Short: "Iniciar o dev server gerenciado (managed mode)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := resolveProxyName(args[0])
			p, err := config.FindProxy(name)
			if err != nil {
				return err
			}
			if !p.Managed {
				return fmt.Errorf("proxy %s não está em managed mode", name)
			}
			if err := proxyops.WriteManagedQuadlet(*p); err != nil {
				return err
			}
			if err := proxyops.StartManaged(name); err != nil {
				return err
			}
			fmt.Printf("Dev server %s iniciado.\n", name)
			return nil
		},
	}
}

func newProxyStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <nome-ou-domínio>",
		Short: "Parar o dev server gerenciado",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := resolveProxyName(args[0])
			if err := proxyops.StopManaged(name); err != nil {
				return err
			}
			fmt.Printf("Dev server %s parado.\n", name)
			return nil
		},
	}
}

func newProxyLogsCmd() *cobra.Command {
	var follow bool
	c := &cobra.Command{
		Use:   "logs <nome-ou-domínio>",
		Short: "Ver logs do dev server gerenciado (journalctl)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := resolveProxyName(args[0])
			unit := proxyops.ManagedUnitName(name) + ".service"
			argv := []string{"--user", "-u", unit, "--no-pager"}
			if follow {
				argv = append(argv, "-f")
			}
			jc := exec.Command("journalctl", argv...)
			jc.Stdout = os.Stdout
			jc.Stderr = os.Stderr
			return jc.Run()
		},
	}
	c.Flags().BoolVarP(&follow, "follow", "f", false, "Seguir logs em tempo real")
	return c
}
