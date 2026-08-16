// allblocks.gsx is T8's fixture (design spec section 11): every stdlib
// block, plus a Custom raw-element subtree and a registered helper call,
// rendered from one sentinel-valued props struct.
package emails

func AllBlocks(props AllBlocksProps) Node {
    return <email.Shell
        wordmark={props.Wordmark}
        shortCode={props.ShortCode}
        tagline={props.Tagline}
        title={props.Title}
        lang="en">
        <email.Signal text={shout(props.ShoutName)} />
        <email.Headline title={props.HeadTitle} lede={props.HeadLede} />
        <email.Panel>
            <email.PanelRow label={props.PanelLabel} value={props.PanelValue} />
        </email.Panel>
        <email.CTA label={props.CTALabel} href={props.CTAHref} />
        <email.PickList title={props.PickTitle}>
            <email.Item>{props.PickItem}</email.Item>
        </email.PickList>
        <email.StatTable title={props.StatTitle} header={props.StatHeader}>
            <Each of={props.StatRows} as="row">
                <email.StatRow cells={row.Cells} mark={row.Mark} />
            </Each>
        </email.StatTable>
        <If cond={props.ShowNote}>
            <email.Note text={props.NoteText} />
        </If>
        <email.Divider />
        <table style="border-collapse: collapse;">
            <tr>
                <td style="vertical-align: top;">{props.TableCell}</td>
            </tr>
        </table>
        <a href={props.LinkHref}>{props.LinkLabel}</a>
        <img src={props.ImgSrc} alt={props.ImgAlt} />
        <email.Footer signoff={props.FooterSign} note={props.FooterNote} />
    </email.Shell>
}
