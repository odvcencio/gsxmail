// newblocks.gsx is the WP5.3 block-corpus fixture: every new component in
// source order, so the sentinel test, the byte goldens, and the
// structural verification pass all exercise the same tree.
package newblocks

func NewBlocks(props NewBlocksProps) Node {
    return <email.Shell
        wordmark={props.Wordmark}
        shortCode={props.ShortCode}
        tagline={props.Tagline}
        title={props.Title}
        lang="en">
        <email.Badge text={props.BadgeNeutralText} />
        <email.Badge text={props.BadgePositiveText} tone="positive" />
        <email.Badge text={props.BadgeWarningText} tone="warning" />
        <email.Badge text={props.BadgeCriticalText} tone="critical" />
        <email.Hero src={props.HeroSrc} alt={props.HeroAlt} width={props.HeroWidth} height={props.HeroHeight} />
        <email.Columns>
            <email.Column
                imgSrc={props.Col1ImgSrc}
                imgAlt={props.Col1ImgAlt}
                imgWidth={props.Col1ImgWidth}
                imgHeight={props.Col1ImgHeight}
                title={props.Col1Title}
                text={props.Col1Text} />
            <email.Column title={props.Col2Title} text={props.Col2Text} />
        </email.Columns>
        <email.Spacer height={props.SpacerHeight} />
        <email.Button variant="primary" label={props.PrimaryLabel} href={props.PrimaryHref} />
        <email.Button variant="secondary" label={props.SecondaryLabel} href={props.SecondaryHref} />
        <email.Button variant="link" label={props.LinkLabel} href={props.LinkHref} width={props.LinkWidth} />
    </email.Shell>
}
