// recap.gsx is the design spec's section 6.5 worked example, the N7
// "YOUR HAUL" draft recap: an <Each>-driven StatTable and a conditional
// <If> note, both wired through gsxmail's dynamic-data pipeline (WP3).
package emails

func DraftRecap(props RecapProps) Node {
    return <email.Shell
        wordmark={props.League}
        shortCode={props.Code}
        tagline={props.Tagline}
        title={props.League}
        lang="en">
        <email.Signal text={"DRAFT COMPLETE // " + props.PickCountLabel} />
        <email.Headline title="THE TAPE IS SEALED." lede={props.Lede} />
        <email.StatTable title="YOUR HAUL //" header={props.HaulHeader}>
            <Each of={props.Haul} as="row">
                <email.StatRow cells={row.Cells} mark={row.IsKeystone} />
            </Each>
        </email.StatTable>
        <If cond={props.HasAutoPicks}>
            <email.Note text={props.AutoPickNote} />
        </If>
        <email.CTA label="SEE THE FULL BOARD →" href={props.BoardURL} />
        <email.Footer signoff="— The Commissioner" note={props.FooterNote} />
    </email.Shell>
}
