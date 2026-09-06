import type {ReactNode} from 'react';
import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  Svg: React.ComponentType<React.ComponentProps<'svg'>>;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'Degrees, not pixels',
    Svg: require('@site/static/img/undraw_docusaurus_mountain.svg').default,
    description: (
      <>
        Positions come back as altitude and azimuth in degrees, paired with an
        angular diameter, plus a time-progress fraction. Screen mapping is a
        separate, optional concern handled by the <code>project</code> package.
      </>
    ),
  },
  {
    title: 'Stateless and concurrency-safe',
    Svg: require('@site/static/img/undraw_docusaurus_tree.svg').default,
    description: (
      <>
        Every call takes the observer and a <code>time.Time</code>; nothing is
        captured at construction. One value serves many callers at once, so a
        service can compute a distinct sky per visitor.
      </>
    ),
  },
  {
    title: 'Meeus, in house',
    Svg: require('@site/static/img/undraw_docusaurus_react.svg').default,
    description: (
      <>
        Jean Meeus&apos; algorithms are implemented in the library itself, with
        no third-party ephemeris dependency. Each one is anchored to a worked
        example from the book and checked against JPL Horizons.
      </>
    ),
  },
];

function Feature({title, Svg, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center">
        <Svg className={styles.featureSvg} role="img" />
      </div>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
