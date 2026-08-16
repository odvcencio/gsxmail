// digest.gsx is the gallery's weekly-digest template (pixel dossier
// section 8.1): a retina Hero, a two-column fluid-hybrid highlights row,
// a StatTable built from <Each>, a Divider, and a static PickList.
package digest

func DigestEmail(props DigestProps) Node {
    return <email.Shell
        wordmark={props.Product}
        shortCode="DIG"
        tagline="WEEKLY DIGEST"
        title={props.Product + " weekly digest"}
        lang="en"
        preheader={props.Period + " — your weekly digest is ready"}>
        <email.Hero src={props.HeroSrc} alt={props.HeroAlt} width={props.HeroWidth} height={props.HeroHeight} />
        <email.Headline title="THIS WEEK." lede={props.Period} />
        <email.Columns>
            <email.Column
                imgSrc={props.Col1ImgSrc}
                imgAlt={props.Col1ImgAlt}
                imgWidth={props.Col1ImgWidth}
                imgHeight={props.Col1ImgHeight}
                title={props.Col1Title}
                text={props.Col1Text} />
            <email.Column
                imgSrc={props.Col2ImgSrc}
                imgAlt={props.Col2ImgAlt}
                imgWidth={props.Col2ImgWidth}
                imgHeight={props.Col2ImgHeight}
                title={props.Col2Title}
                text={props.Col2Text} />
        </email.Columns>
        <email.Divider />
        <email.StatTable title="BY THE NUMBERS //" header={props.StatHeader}>
            <Each of={props.Stats} as="row">
                <email.StatRow cells={row.Cells} />
            </Each>
        </email.StatTable>
        <email.PickList title="ALSO THIS WEEK">
            <email.Item>Read the top story</email.Item>
            <email.Item>Catch up on the comment threads</email.Item>
            <email.Item>Check this week's leaderboard</email.Item>
        </email.PickList>
    </email.Shell>
}
