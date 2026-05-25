package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/proxyops"
	"github.com/spf13/cobra"
)

// NewProxyCmd is the root of `lerd proxy …`.
func NewProxyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Gerenciar proxies para projetos não-PHP (SPAs, dev servers)",
		Long:  "Registra um domínio local que faz reverse proxy para um dev server arbitrário no host (Vue/Quasar/Nuxt/Vite). HTTPS e WebSocket habilitados por padrão.",
	}
	cmd.AddCommand(newProxyAddCmd())
	cmd.AddCommand(newProxyLsCmd())
	cmd.AddCommand(newProxyRmCmd())
	cmd.AddCommand(newProxySecureCmd(true))
	cmd.AddCommand(newProxySecureCmd(false))
	cmd.AddCommand(newProxyPauseCmd(true))
	cmd.AddCommand(newProxyPauseCmd(false))
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

	c := &cobra.Command{
		Use:   "add <domínio>",
		Short: "Registrar um novo proxy",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := proxyops.Add(proxyops.AddOptions{
				Domain:      strings.ToLower(args[0]),
				Port:        port,
				Path:        path,
				NoSecure:    noSecure,
				Managed:     managed,
				Command:     cmdStr,
				NodeVersion: nodeVersion,
				AutoStart:   autostart,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Proxy criado: %s → %s:%d (secured=%v, managed=%v)\n",
				p.PrimaryDomain(), upstreamForDisplay(p), p.UpstreamPort, p.Secured, p.Managed)
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
	_ = c.MarkFlagRequired("port")
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
