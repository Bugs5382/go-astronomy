import {recommendedThemeConfig} from '@the-rabbit-hole/docs-theme/config';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

const organizationName = 'Bugs5382';
const projectName = 'go-astronomy';
const repoUrl = `https://github.com/${organizationName}/${projectName}`;

const config: Config = {
  title: 'go-astronomy',
  tagline:
    'Observer-aware Sun, Moon, star, and constellation positions for Go',
  favicon: 'img/favicon.svg',

  // Future flags, see https://docusaurus.io/docs/api/docusaurus-config#future
  future: {
    v4: true, // Improve compatibility with the upcoming Docusaurus v4
  },

  // GitHub Pages project-site hosting: https://Bugs5382.github.io/go-astronomy/
  url: `https://${organizationName}.github.io`,
  baseUrl: `/${projectName}/`,

  organizationName,
  projectName,
  trailingSlash: false,

  onBrokenLinks: 'throw',

  markdown: {
    // Parse .md as CommonMark and .mdx as MDX, so hand-written Markdown with
    // angle brackets and braces does not need MDX escaping.
    format: 'detect',
    hooks: {
      onBrokenMarkdownLinks: 'throw',
    },
  },

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang.
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: 'docs',
          editUrl: `${repoUrl}/tree/main/website/`,
        },
        blog: false,
        theme: {
          // The brand stylesheet is the only global CSS this site loads. A
          // site stylesheet layered on top of it would win over the brand
          // tokens, which is how the scaffold's green and teal survived a
          // theme that was supposedly already wired in.
          customCss: require.resolve(
            '@the-rabbit-hole/docs-theme/styles/custom.css',
          ),
        },
      } satisfies Preset.Options,
    ],
  ],

  // Registers the theme so its swizzled components resolve through `@theme`;
  // today that is the collapsible right-side table of contents.
  plugins: ['@the-rabbit-hole/docs-theme'],

  themeConfig: {
    // Brand defaults: dark-first colour mode, hideable docs sidebar, prism.
    ...recommendedThemeConfig,
    image: 'img/docusaurus-social-card.jpg',
    navbar: {
      title: 'go-astronomy',
      logo: {
        alt: 'go-astronomy logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'Docs',
        },
        {
          href: 'https://pkg.go.dev/github.com/Bugs5382/go-astronomy',
          label: 'API reference',
          position: 'right',
        },
        {
          href: repoUrl,
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            {label: 'Overview', to: '/docs/intro'},
            {label: 'Getting started', to: '/docs/getting-started'},
            {label: 'Concepts', to: '/docs/concepts'},
          ],
        },
        {
          title: 'Reference',
          items: [
            {
              label: 'pkg.go.dev',
              href: 'https://pkg.go.dev/github.com/Bugs5382/go-astronomy',
            },
            {label: 'Error codes', to: '/docs/reference/errors'},
          ],
        },
        {
          title: 'More',
          items: [{label: 'GitHub', href: repoUrl}],
        },
      ],
      copyright: `Copyright ${new Date().getFullYear()} Shane. Built with Docusaurus.`,
    },
    prism: {
      ...recommendedThemeConfig.prism,
      additionalLanguages: [
        ...recommendedThemeConfig.prism.additionalLanguages,
        'go',
      ],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
