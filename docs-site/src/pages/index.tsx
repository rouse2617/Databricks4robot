import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Heading from '@theme/Heading';
import styles from './index.module.css';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <Heading as="h1" className="hero__title">
          {siteConfig.title}
        </Heading>
        <p className="hero__subtitle">{siteConfig.tagline}</p>
        <div className={styles.buttons}>
          <Link
            className="button button--secondary button--lg"
            to="/intro">
            开始阅读文档 →
          </Link>
        </div>
      </div>
    </header>
  );
}

export default function Home(): ReactNode {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title={`${siteConfig.title}`}
      description="Cyber Databrew 数据流水线平台文档">
      <HomepageHeader />
      <main>
        <div className="container" style={{padding: '3rem 0'}}>
          <div className="row">
            <div className="col col--4">
              <div className="card" style={{marginBottom: '1rem'}}>
                <div className="card__body">
                  <h3>🚀 快速开始</h3>
                  <p>5 分钟上手 SDK，完成首次 API 调用。</p>
                  <Link to="/getting-started/quickstart">开始 →</Link>
                </div>
              </div>
            </div>
            <div className="col col--4">
              <div className="card" style={{marginBottom: '1rem'}}>
                <div className="card__body">
                  <h3>📖 核心概念</h3>
                  <p>了解资产、流水线、交付和标签的核心数据模型。</p>
                  <Link to="/core-concepts/assets">探索 →</Link>
                </div>
              </div>
            </div>
            <div className="col col--4">
              <div className="card" style={{marginBottom: '1rem'}}>
                <div className="card__body">
                  <h3>🔌 API 参考</h3>
                  <p>完整的 REST API 端点列表和使用方法。</p>
                  <Link to="/api/overview">查看 →</Link>
                </div>
              </div>
            </div>
          </div>
        </div>
      </main>
    </Layout>
  );
}
