import type {ReactNode} from 'react';
import Landing from '@the-rabbit-hole/docs-theme/landing';

const repoUrl = 'https://github.com/Bugs5382/go-astronomy';
const goDocUrl = 'https://pkg.go.dev/github.com/Bugs5382/go-astronomy';

// Card glyphs are written as escaped codepoints rather than literal emoji: the
// repository's hygiene gate allows emoji in Markdown and blocks them in
// source. The rendered card is identical either way.
const RULER = '\u{1F4D0}';
const THREAD = '\u{1F9F5}';
const BOOK = '\u{1F4D6}';

const quickstart = `go get github.com/Bugs5382/go-astronomy

// Where is the Sun, for this observer, right now?
pos, err := sun.PositionAt(observer, time.Now())`;

export default function Home(): ReactNode {
  return (
    <Landing
      buttons={[
        {label: 'Get started', to: '/docs/getting-started', variant: 'secondary'},
        {label: 'Documentation', to: '/docs/intro'},
        {label: 'API reference', href: goDocUrl},
        {label: 'GitHub', href: repoUrl},
      ]}
      features={[
        {
          icon: RULER,
          title: 'Degrees, not pixels',
          body: (
            <>
              Positions come back as altitude and azimuth in degrees, paired
              with an angular diameter, plus a time-progress fraction. Screen
              mapping is a separate, optional concern handled by the{' '}
              <code>project</code> package.
            </>
          ),
        },
        {
          icon: THREAD,
          title: 'Stateless and concurrency-safe',
          body: (
            <>
              Every call takes the observer and a <code>time.Time</code>;
              nothing is captured at construction. One value serves many callers
              at once, so a service can compute a distinct sky per visitor.
            </>
          ),
        },
        {
          icon: BOOK,
          title: 'Meeus, in house',
          body: (
            <>
              Jean Meeus&apos; algorithms are implemented in the library itself,
              with no third-party ephemeris dependency. Each one is anchored to
              a worked example from the book and checked against JPL Horizons.
            </>
          ),
        },
      ]}
      quickstart={{
        code: quickstart,
        cta: {
          label: 'Read the docs',
          to: '/docs/intro',
          variant: 'primary',
        },
        language: 'go',
        lede: 'Add the module, then ask for a position.',
        title: 'Quickstart',
      }}
    />
  );
}
