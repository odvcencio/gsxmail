// alert.gsx is the gallery's notification template (pixel dossier section
// 8.1): Shell, Signal, Badge, Note, Button. Severity renders as the
// Signal line, a bordered Badge, and the Note's own structural mark
// (border-left plus background) — never color alone.
package alert

func AlertEmail(props AlertProps) Node {
    return <email.Shell
        wordmark={props.Product}
        shortCode="ALT"
        tagline="SYSTEM ALERTS"
        title={props.Product + " alert"}
        lang="en"
        preheader={props.Severity + ": " + props.Message}>
        <email.Signal text={"SYSTEM ALERT // " + props.Product} />
        <If cond={props.IsCritical}>
            <email.Badge text={props.Severity} tone="critical" />
        </If>
        <If cond={!props.IsCritical}>
            <email.Badge text={props.Severity} tone="warning" />
        </If>
        <email.Note text={props.Message} />
        <email.Button variant="primary" label={props.ActionLabel} href={props.ActionURL} />
    </email.Shell>
}
